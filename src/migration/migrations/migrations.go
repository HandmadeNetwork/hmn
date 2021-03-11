package migrations

import "git.handmade.network/hmn/hmn/src/migration/types"

var All map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)

func registerMigration(m types.Migration) {
	All[m.Version()] = m
}
