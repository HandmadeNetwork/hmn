package assets

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var client *s3.Client

func init() {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				config.Config.DigitalOcean.AssetsSpacesKey,
				config.Config.DigitalOcean.AssetsSpacesSecret,
				"",
			),
		),
		awsconfig.WithRegion(config.Config.DigitalOcean.AssetsSpacesRegion),
		awsconfig.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: config.Config.DigitalOcean.AssetsSpacesEndpoint,
			}, nil
		})),
	)
	if err != nil {
		panic(err)
	}
	client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
}

type CreateInput struct {
	Content     []byte
	Filename    string
	ContentType string

	// Optional params
	UploaderID    *int // HMN user id
	Width, Height int
}

var REIllegalFilenameChars = regexp.MustCompile(`[^\w\-.]`)

func SanitizeFilename(filename string) string {
	if filename == "" {
		return "unnamed"
	}
	return REIllegalFilenameChars.ReplaceAllString(filename, "_")
}

func AssetKey(id, filename string) string {
	return fmt.Sprintf("%s/%s", id, filename)
}

type InvalidAssetError error

func Create(ctx context.Context, dbConn db.ConnOrTx, in CreateInput) (*models.Asset, error) {
	filename := SanitizeFilename(in.Filename)

	if len(in.Content) == 0 {
		return nil, InvalidAssetError(fmt.Errorf("could not upload asset '%s': no bytes of data were provided", filename))
	}
	if in.ContentType == "" {
		return nil, InvalidAssetError(fmt.Errorf("could not upload asset '%s': no content type provided", filename))
	}

	// Upload the asset to the DO space
	id := uuid.New()
	key := AssetKey(id.String(), filename)
	checksum := fmt.Sprintf("%x", sha1.Sum(in.Content))

	upload := func() error {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      &config.Config.DigitalOcean.AssetsSpacesBucket,
			Key:         &key,
			Body:        bytes.NewReader(in.Content),
			ACL:         types.ObjectCannedACLPublicRead,
			ContentType: &in.ContentType,
		})
		return err
	}

	err := upload()
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) && apiError.ErrorCode() == "NoSuchBucket" {
			_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: &config.Config.DigitalOcean.AssetsSpacesBucket,
			})
			if err != nil {
				return nil, oops.New(err, "failed to create assets bucket")
			}

			err = upload()
			if err != nil {
				return nil, oops.New(err, "failed to upload asset")
			}
		} else {
			return nil, oops.New(err, "failed to upload asset")
		}
	}

	var thumbnailKey *string

	previewBytes, err := ExtractPreview(ctx, in.ContentType, in.Content)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to generate preview for asset")
	} else if len(previewBytes) > 0 {
		keyStr := AssetKey(id.String(), fmt.Sprintf("%s_thumb.png", id.String()))
		thumbnailType := "image/png"
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      &config.Config.DigitalOcean.AssetsSpacesBucket,
			Key:         &keyStr,
			Body:        bytes.NewReader(previewBytes),
			ACL:         types.ObjectCannedACLPublicRead,
			ContentType: &thumbnailType,
		})
		if err != nil {
			logging.Error().Err(err).Msg("Failed to upload thumbnail for video")
		} else {
			thumbnailKey = &keyStr
		}
	}

	// Save a record in our database
	// TODO(db): Would be convient to use RETURNING here...
	_, err = dbConn.Exec(ctx,
		`
		INSERT INTO asset (id, s3_key, thumbnail_s3_key, filename, size, mime_type, sha1sum, width, height, uploader_id)
		VALUES            ($1, $2,     $3,               $4,       $5,   $6,        $7,      $8,    $9,     $10)
		`,
		id,
		key,
		thumbnailKey,
		filename,
		len(in.Content),
		in.ContentType,
		checksum,
		in.Width,
		in.Height,
		in.UploaderID,
	)
	if err != nil {
		return nil, oops.New(err, "failed to save asset record")
	}

	// Fetch and return the new record
	asset, err := db.QueryOne[models.Asset](ctx, dbConn,
		`
		SELECT $columns
		FROM asset
		WHERE id = $1
		`,
		id,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch newly-created asset")
	}

	return asset, nil
}

func ExtractPreview(ctx context.Context, mimeType string, inBytes []byte) ([]byte, error) {
	if config.Config.PreviewGeneration.FFMpegPath == "" {
		return nil, nil
	}

	if !strings.HasPrefix(mimeType, "video") {
		return nil, nil
	}

	file, err := os.CreateTemp("", "hmnasset")
	if err != nil {
		return nil, oops.New(err, "Failed to create temp file for preview generation")
	}
	defer os.Remove(file.Name())
	_, err = file.Write(inBytes)
	if err != nil {
		return nil, oops.New(err, "Failed to write to temp file for preview generation")
	}
	err = file.Close()
	if err != nil {
		return nil, oops.New(err, "Failed to close temp file for preview generation")
	}

	args := fmt.Sprintf("-i %s -filter_complex [0]select=gte(n\\,1)[s0] -map [s0] -f image2 -vcodec png -vframes 1 pipe:1", file.Name())
	execPath := config.Config.PreviewGeneration.FFMpegPath
	if config.Config.PreviewGeneration.CPULimitPath != "" {
		args = fmt.Sprintf("-l 10 -- %s %s", execPath, args)
		execPath = config.Config.PreviewGeneration.CPULimitPath
	}
	ffmpegCmd := exec.CommandContext(ctx, execPath, strings.Split(args, " ")...)

	var output bytes.Buffer
	var errorOut bytes.Buffer
	ffmpegCmd.Stdout = &output
	ffmpegCmd.Stderr = &errorOut
	err = ffmpegCmd.Run()
	if err != nil {
		logging.Error().Str("ffmpeg output", string(errorOut.Bytes())).Msg("FFMpeg returned error while generating preview thumbnail")
		return nil, oops.New(err, "FFMpeg failed for preview generation")
	}

	return output.Bytes(), nil
}

func BackgroundPreviewGeneration(ctx context.Context, conn *pgxpool.Pool) jobs.Job {
	log := logging.ExtractLogger(ctx).With().Str("module", "preview_gen").Logger()
	job := jobs.New()

	go func() {
		defer job.Done()
		log.Debug().Msg("Starting preview gen job")
		assets, err := db.Query[models.Asset](ctx, conn,
			`
			SELECT $columns
			FROM asset
			WHERE mime_type LIKE 'video%' AND (thumbnail_s3_key IS NULL OR thumbnail_s3_key = '')
			`,
		)
		if err != nil {
			log.Error().Err(oops.New(err, "Failed to fetch assets for preview generation")).Msg("Asset preview generation job failed")
			return
		}

		log.Debug().Int("Num assets", len(assets)).Msg("Processing...")

		for _, asset := range assets {
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Debug().Str("AssetID", asset.ID.String()).Msg("Generating preview")
			assetUrl := hmnurl.BuildS3Asset(asset.S3Key)
			resp, err := http.Get(assetUrl)
			if err != nil || resp.StatusCode != 200 {
				log.Error().Err(err).Msg("Failed to fetch asset file for preview generation")
				continue
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error().Err(err).Msg("Failed to read asset body for preview generation")
				continue
			}
			thumbBytes, err := ExtractPreview(ctx, asset.MimeType, body)
			if err != nil {
				log.Error().Err(err).Msg("Failed to run extraction for preview generation")
				continue
			} else if len(thumbBytes) > 0 {
				keyStr := AssetKey(asset.ID.String(), fmt.Sprintf("%s_thumb.png", asset.ID.String()))
				thumbnailType := "image/png"
				_, err = client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:      &config.Config.DigitalOcean.AssetsSpacesBucket,
					Key:         &keyStr,
					Body:        bytes.NewReader(thumbBytes),
					ACL:         types.ObjectCannedACLPublicRead,
					ContentType: &thumbnailType,
				})
				if err != nil {
					log.Error().Err(err).Msg("Failed to upload thumbnail for video")
					continue
				}

				_, err = conn.Exec(ctx,
					`
					UPDATE asset
					SET thumbnail_s3_key = $1
					WHERE asset.id = $2
					`,
					keyStr,
					asset.ID,
				)
				if err != nil {
					log.Error().Err(err).Msg("Failed to update asset for preview generation")
					continue
				}
				log.Debug().Str("AssetID", asset.ID.String()).Msg("Generated preview successfully!")
			}
		}
		log.Debug().Msg("No more previews to generate")
	}()

	return job
}
