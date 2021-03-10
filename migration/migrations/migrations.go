package migrations

import "git.handmade.network/hmn/hmn/migration/types"

var All map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)

func registerMigration(m types.Migration) {
	All[m.Version()] = m
}
