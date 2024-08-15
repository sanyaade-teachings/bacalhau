package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// V3Migration updates the repo replacing repo.version and update.json with system_metadata.yaml.
// It does the following:
// - creates system_metadata.yaml with repo version 4.
// - sets the last update check time in system_metadata.yaml to unix time zero.
// - if an installationID is present in the config its value is persisted to system_metadata.yaml.
// - removes update.json if the file is present.
// - removes repo.version.
// - creates a new directory .bacalhau/compute_store/executions which contains the content of ./bacalhau/execution_store
// - removes ./bacalhau/execution_store
// - iff a user has configured a user key path, the configured value is copied to .bacalhau/user_id.pem
// - iff a user has configured a auth tokens path, the configured value is copied to .bacalhau/tokens.json
var V3Migration = StagedMigration(
	repo.Version3,
	repo.Version4,
	func(r repo.FsRepo) error {
		repoPath, err := r.Path()
		if err != nil {
			return err
		}
		_, fileCfg, err := readConfig(r)
		if err != nil {
			return err
		}
		// Initialize the SystemMetadataFile in the staging directory
		if err := r.WriteVersion(repo.Version4); err != nil {
			return err
		}
		if err := r.WriteLastUpdateCheck(time.UnixMilli(0)); err != nil {
			return err
		}
		if fileCfg.User.InstallationID != "" {
			if err := r.WriteInstallationID(fileCfg.User.InstallationID); err != nil {
				return err
			}
		}

		// ignore this error as the file may not exist
		_ = os.Remove(filepath.Join(repoPath, "update.json"))
		if err := os.Remove(filepath.Join(repoPath, repo.LegacyVersionFile)); err != nil {
			return fmt.Errorf("removing legacy repo version file: %w", err)
		}

		if fileCfg.User.KeyPath != "" {
			if err := copyFile(fileCfg.User.KeyPath, filepath.Join(repoPath, repo.UserKeyFile)); err != nil {
				return fmt.Errorf("copying user key file: %w", err)
			}
		}

		if fileCfg.Auth.TokensPath != "" {
			if err := copyFile(fileCfg.Auth.TokensPath, filepath.Join(repoPath, repo.AuthTokensFile)); err != nil {
				return fmt.Errorf("copying auth tokens file: %w", err)
			}
		}

		if fileCfg.Node.Compute.ExecutionStore.Path != "" {
			if err := copyFile(
				fileCfg.Node.Compute.ExecutionStore.Path,
				filepath.Join(repoPath, repo.ComputeDirKey, "executions.db"),
			); err != nil {
				return fmt.Errorf("copying execution database: %w", err)
			}
		}

		if fileCfg.Node.Requester.JobStore.Path != "" {
			if err := copyFile(
				fileCfg.Node.Requester.JobStore.Path,
				filepath.Join(repoPath, repo.OrchestratorDirKey, "jobs.db"),
			); err != nil {
				return fmt.Errorf("copying job database: %w", err)
			}
		}

		from := fileCfg.Node.ComputeStoragePath
		if from == "" {
			from = filepath.Join(repoPath, "executor_storages")
		}
		to := filepath.Join(repoPath, repo.ExecutionDirKey)
		log.Info().Str("from", from).Str("to", to).Msg("copying executor storages")
		if err := copyFS(to, os.DirFS(from)); err != nil {
			return fmt.Errorf("copying executor storages: %w", err)
		}
		if err := os.RemoveAll(from); err != nil {
			return err
		}

		return nil
	},
)
