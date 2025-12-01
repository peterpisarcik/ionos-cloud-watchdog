package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/config"
	"github.com/spf13/cobra"
)

var (
	initToken      string
	initUsername   string
	initPassword   string
	initKubeconfig string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long: `Initialize configuration file interactively or using flags.

Examples:
  # Interactive mode
  ionos-cloud-watchdog config init

  # Using flags (avoids paste issues)
  ionos-cloud-watchdog config init --token "your-token" --kubeconfig ~/.kube/config
  ionos-cloud-watchdog config init --username user --password pass`,
	RunE: handleConfigInit,
}

func init() {
	configInitCmd.Flags().StringVar(&initToken, "token", "", "IONOS Cloud API token")
	configInitCmd.Flags().StringVar(&initUsername, "username", "", "IONOS Cloud username (alternative to token)")
	configInitCmd.Flags().StringVar(&initPassword, "password", "", "IONOS Cloud password (alternative to token)")
	configInitCmd.Flags().StringVar(&initKubeconfig, "kubeconfig", "", "path to kubeconfig file")

	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func handleConfigInit(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing config file...")

	cfg := &config.Config{}

	if initToken != "" || initUsername != "" {
		cfg.IONOS.Token = initToken
		cfg.IONOS.Username = initUsername
		cfg.IONOS.Password = initPassword
		cfg.Kubeconfig = initKubeconfig
	} else {
		if err := promptForConfig(cfg); err != nil {
			return err
		}
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("\nConfiguration saved to: %s\n", configPath)
	fmt.Println("You can now run ionos-cloud-watchdog without setting environment variables.")

	return nil
}

func promptForConfig(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("IONOS Cloud API Token (leave empty to use username/password): ")
	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading token: %w", err)
	}
	cfg.IONOS.Token = strings.TrimSpace(token)

	if cfg.IONOS.Token == "" {
		fmt.Print("IONOS Cloud Username: ")
		username, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading username: %w", err)
		}
		cfg.IONOS.Username = strings.TrimSpace(username)

		fmt.Print("IONOS Cloud Password: ")
		password, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading password: %w", err)
		}
		cfg.IONOS.Password = strings.TrimSpace(password)
	}

	fmt.Print("Kubeconfig path (leave empty for default ~/.kube/config): ")
	kubeconfigInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading kubeconfig path: %w", err)
	}
	cfg.Kubeconfig = strings.TrimSpace(kubeconfigInput)

	return nil
}
