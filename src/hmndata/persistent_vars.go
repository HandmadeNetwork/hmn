package hmndata

import (
	"context"
	"encoding/json"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

type PersistentVarName string

const (
	VarNameDiscordLivestreamMessage PersistentVarName = "discord_livestream_message"
)

type StreamDetails struct {
	Username  string    `json:"username"`
	StartTime time.Time `json:"start_time"`
	Title     string    `json:"title"`
}

type DiscordLivestreamMessage struct {
	MessageID string          `json:"message_id"`
	Streamers []StreamDetails `json:"streamers"`
}

// NOTE(asaf): Returns db.NotFound if the variable isn't in the db.
func FetchPersistentVar[T any](
	ctx context.Context,
	dbConn db.ConnOrTx,
	varName PersistentVarName,
) (*T, error) {
	persistentVar, err := db.QueryOne[models.PersistentVar](ctx, dbConn,
		`
		SELECT $columns
			FROM persistent_var
		WHERE name = $1
		`,
		varName,
	)

	if err != nil {
		return nil, err
	}

	jsonString := persistentVar.Value
	var result T
	err = json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal persistent var value")
	}

	return &result, nil
}

func StorePersistentVar[T any](
	ctx context.Context,
	dbConn db.ConnOrTx,
	name PersistentVarName,
	value *T,
) error {
	jsonString, err := json.Marshal(value)
	if err != nil {
		return oops.New(err, "failed to marshal variable")
	}

	_, err = dbConn.Exec(ctx,
		`
		INSERT INTO persistent_var (name, value) 
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET
			value = EXCLUDED.value
		`,
		name,
		jsonString,
	)

	if err != nil {
		return oops.New(err, "failed to insert var to db")
	}

	return nil
}

func RemovePersistentVar(ctx context.Context, dbConn db.ConnOrTx, name PersistentVarName) error {
	_, err := dbConn.Exec(ctx,
		`
		DELETE FROM persistent_var
		WHERE name = $1
		`,
		name,
	)
	if err != nil {
		return oops.New(err, "failed to delete var")
	}
	return nil
}
