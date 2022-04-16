package assets

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"regexp"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
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
	return fmt.Sprintf("%s%s/%s", config.Config.DigitalOcean.AssetsPathPrefix, id, filename)
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

	// Save a record in our database
	// TODO(db): Would be convient to use RETURNING here...
	_, err = dbConn.Exec(ctx,
		`
		INSERT INTO handmade_asset (id, s3_key, filename, size, mime_type, sha1sum, width, height, uploader_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
		id,
		key,
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
		FROM handmade_asset
		WHERE id = $1
		`,
		id,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch newly-created asset")
	}

	return asset, nil
}
