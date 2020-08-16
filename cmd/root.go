package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:                   "runtainer [runtainer flags] image [backend flags] [-- [in container args]]",
		Short:                 "Run anything as Container",
		Long:                  "See https://github.com/plumber-cd/runtainer for details",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			host.DiscoverHost()

			imageName := args[0]
			image.DiscoverImage(imageName)
			inArgs := args[1:]

			volumes.DiscoverVolumes()

			allSettings, err := json.MarshalIndent(viper.AllSettings(), "", "  ")
			if err != nil {
				log.Error.Fatal(err)
			}
			log.Debug.Printf("Settings: %s", string(allSettings))

			if viper.GetBool("kube") {
				runInKube(inArgs)
			} else {
				runInDocker(inArgs)
			}
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.runtainer.yaml)")

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose mode")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	rootCmd.PersistentFlags().BoolP("kube", "k", false, "Use kubectl as backend instead of docker")
	viper.BindPFlag("kube", rootCmd.PersistentFlags().Lookup("kube"))

	rootCmd.PersistentFlags().BoolP("stdin", "i", true, "Use --interactive for docker and --stdin for kubectl")
	viper.BindPFlag("stdin", rootCmd.PersistentFlags().Lookup("stdin"))

	rootCmd.PersistentFlags().BoolP("tty", "t", true, "Use --tty for backend, disable if piping output into some other stdin")
	viper.BindPFlag("tty", rootCmd.PersistentFlags().Lookup("tty"))

	rootCmd.PersistentFlags().StringP("dir", "d", "", "Use different folder to make a CWD in the container (default is the host CWD)")
	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))

	rootCmd.Flags().SetInterspersed(false)
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if cfgFile != "" {
		if !fileExists(cfgFile) {
			log.Error.Fatalf("Config file not found: %s", cfgFile)
		}
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Error.Fatal(err)
		}

		// Search config in home directory with name ".runtainer" (without extension).
		viper.SetConfigName("config")
		viper.AddConfigPath(home + "/.runtainer")
	}

	viper.SetEnvPrefix("RT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Debug.Print("Using global config file:", viper.ConfigFileUsed())
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Error.Fatal(err)
	}
	viper.SetConfigName(".runtainer")
	viper.AddConfigPath(cwd)

	if err := viper.MergeInConfig(); err == nil {
		log.Debug.Print("Using local config file:", viper.ConfigFileUsed())
	}

	// Re-initialize loggers in case output settings changed
	log.SetupLog()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func splitArgs(args ...string) ([]string, []string) {
	log.Debug.Printf("args: %s", strings.Join(args, " "))
	backendArgs := args
	var containerArgs []string
	for i := range args {
		if args[i] == "--" {
			backendArgs = args[:i]
			containerArgs = args[(i + 1):]
			break
		}
	}
	log.Debug.Printf("backendArgs: %s", strings.Join(backendArgs, " "))
	log.Debug.Printf("containerArgs: %s", strings.Join(containerArgs, " "))
	return backendArgs, containerArgs
}

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		log.Error.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}
