package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	tmtypes "github.com/tendermint/tendermint/types"
)

const flagGenesisTime = "genesis-time"

// MigrateToV2 migrates the state from v1 to v2.
func MigrateToV2(appState types.AppMap) types.AppMap {
	return appState
}

// Allow applications to extend and modify the migration process.
//
// Ref: https://github.com/cosmos/cosmos-sdk/issues/5041
var migrationMap = types.MigrationMap{
	"v2.0.0": MigrateToV2,
}

// GetMigrationCallback returns a MigrationCallback for a given version.
func GetMigrationCallback(version string) types.MigrationCallback {
	return migrationMap[version]
}

// GetMigrationVersions get all migration version in a sorted slice.
func GetMigrationVersions() []string {
	versions := make([]string, len(migrationMap))

	var i int

	for version := range migrationMap {
		versions[i] = version
		i++
	}

	sort.Strings(versions)

	return versions
}

// MigrateGenesisCmd returns a command to execute genesis state migration.
func MigrateGenesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [target-version] [genesis-file]",
		Short: "Migrate genesis to a specified target version",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			target := args[0]
			importGenesis := args[1]

			genDoc, err := tmtypes.GenesisDocFromFile(importGenesis)
			if err != nil {
				return fmt.Errorf("failed to read genesis document from file %s: %w", importGenesis, err)
			}

			var initialState types.AppMap
			if err := json.Unmarshal(genDoc.AppState, &initialState); err != nil {
				return fmt.Errorf("failed to JSON unmarshal initial genesis state: %w", err)
			}

			migrationFunc := GetMigrationCallback(target)
			if migrationFunc == nil {
				return fmt.Errorf("unknown migration function for version: %s", target)
			}

			// TODO: handler error from migrationFunc call
			newGenState := migrationFunc(initialState)

			genDoc.AppState, err = json.Marshal(newGenState)
			if err != nil {
				return fmt.Errorf("failed to JSON marshal migrated genesis state: %w", err)
			}

			genesisTime, _ := cmd.Flags().GetString(flagGenesisTime)
			if genesisTime != "" {
				var t time.Time

				err := t.UnmarshalText([]byte(genesisTime))
				if err != nil {
					return fmt.Errorf("failed to unmarshal genesis time: %w", err)
				}

				genDoc.GenesisTime = t
			}

			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			if chainID != "" {
				genDoc.ChainID = chainID
			}

			bz, err := json.MarshalIndent(genDoc, "", " ")
			if err != nil {
				return fmt.Errorf("failed to marshal genesis doc: %w", err)
			}

			sortedBz, err := sdk.SortJSON(bz)
			if err != nil {
				return fmt.Errorf("failed to sort JSON genesis doc: %w", err)
			}

			fmt.Println(string(sortedBz))
			return nil
		},
	}

	cmd.Flags().String(flagGenesisTime, "", "override genesis_time with this flag")
	cmd.Flags().String(flags.FlagChainID, "", "override chain_id with this flag")

	return cmd
}
