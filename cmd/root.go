package cmd

import (
	"encoding/json"
	llog "log"
	"os"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/plumber-cd/runtainer/backends/k8s"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:                   "runtainer [runtainer flags] image [container cmd] [-- [container args]]",
		Short:                 "Run anything as a Container",
		Long:                  "See https://github.com/plumber-cd/runtainer/README.md for details",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.Debug.Print("Start root command execution")

			// the args will contain all the args unrecognized by cobra after the first positional arg (not dash prefixed)
			// the first not dash prefixed arg must be the image name
			imageName := args[0]
			log.Debug.Printf("Image: %s", imageName)

			// rest of the args split by -- delimiter
			// See POSIX chapter 12.02, Guideline 10: https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html#tag_12_02
			// On the left, args considered to be passed to the backend (docker/kubectl/etc), on the right args considered to be passed to the container
			containerCmd, containerArgs := splitArgs(args[1:])

			// run discovery routines that will publish all the facts to viper for backend engine to interpret
			discover(imageName)

			// just for debugging, dump full viper data before passing it to the backends
			allSettings, err := json.MarshalIndent(viper.AllSettings(), "", "  ")
			if err != nil {
				log.Normal.Panic(err)
			}
			log.Debug.Printf("Settings: %s", string(allSettings))

			k8s.Run(containerCmd, containerArgs)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "global config file (default is $HOME/.runtainer.yaml)")

	rootCmd.PersistentFlags().BoolP("quiet", "q", false, `Enable quiet mode.
	By default runtainer never prints to StdOut,
	reserving that channel exclusively to the container.
	But it does print messages to StdErr.
	Enabling quiet mode will redirect all messages to the info logger.
	If --log mode was not enabled - these messages will be discarded.`)
	if err := viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet")); err != nil {
		llog.Panic(err)
	}

	rootCmd.PersistentFlags().Bool("log", false, "Enables info logs to file")
	if err := viper.BindPFlag("log", rootCmd.PersistentFlags().Lookup("log")); err != nil {
		llog.Panic(err)
	}

	rootCmd.PersistentFlags().Bool("debug", false, "Enables info and debug logs to file")
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		llog.Panic(err)
	}

	rootCmd.PersistentFlags().BoolP("interactive", "i", true, `Disable to not to attach to the container.
	By default we wait till pod becomes Running and then - attaching to it.
	If container expected to run a script in non-interactive mode and exit,
	- the tool might try to attach to the container that is already finished and fail.
	Disable interactive mode in this case - then it will not attempt to attach
	and instead will just stream logs until containe becomes either Succeeded or Failed.
	This automatically disables --stdin and --tty.`)
	if err := viper.BindPFlag("interactive", rootCmd.PersistentFlags().Lookup("interactive")); err != nil {
		llog.Panic(err)
	}

	rootCmd.PersistentFlags().BoolP("stdin", "s", true, "Redirect host StdIn to the container")
	if err := viper.BindPFlag("stdin", rootCmd.PersistentFlags().Lookup("stdin")); err != nil {
		llog.Panic(err)
	}

	rootCmd.PersistentFlags().BoolP("tty", "t", true, "Enable TTY, disable if piping something to stdin")
	if err := viper.BindPFlag("tty", rootCmd.PersistentFlags().Lookup("tty")); err != nil {
		llog.Panic(err)
	}

	rootCmd.PersistentFlags().StringP("dir", "d", "", "Use different folder to make a CWD in the container (default is the host CWD)")
	if err := viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir")); err != nil {
		llog.Panic(err)
	}

	rootCmd.PersistentFlags().Bool("dry-run", false, "Dry Run mode will not execute the container, only print to StdOut a pod spec it would have run.")
	if err := viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run")); err != nil {
		llog.Panic(err)
	}

	rootCmd.Flags().SetInterspersed(false)
}

func initConfig() {
	log.Debug.Print("Read viper configs")

	if cfgFile != "" {
		log.Debug.Printf("Using custom global config path: %s", cfgFile)

		exists, err := utils.FileExists(cfgFile)
		if err != nil {
			log.Normal.Panic(err)
		}
		if !exists {
			log.Normal.Fatalf("Global config file not found: %s", cfgFile)
		}

		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		log.Debug.Print("Using default global config in user home")

		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Error.Panic(err)
		}

		// Search config in home directory with name ".runtainer" (without extension).
		viper.SetConfigName("config")
		viper.AddConfigPath(filepath.Join(home, ".runtainer"))
	}

	log.Debug.Print("Enabling ENV parsing for viper")
	// This is so we can set any nested viper settings via env variables, replacing every . with _
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// RT short from RunTainer
	viper.SetEnvPrefix("RT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.Debug.Printf("Global %s, skipping...", err)
		default:
			log.Error.Panic(err)
		}
	} else {
		log.Debug.Print("Using global config file:", viper.ConfigFileUsed())
	}

	// try to read (if exists) local config file in the cwd
	cwd, err := os.Getwd()
	if err != nil {
		log.Error.Panic(err)
	}
	readLocalConfig(cwd)

	// by the time we load viper configs, we haven't discovered the host yet,
	// so we couldn't use host.Cwd above
	// but just in case if user provided some custom directory, check for possible config there too
	if d := viper.GetString("dir"); d != "" {
		readLocalConfig(d)
	}

	log.Debug.Print("Viper configs loaded, re-initialize logger in case anything changed...")
	log.SetupLog()
}

// readLocalConfig read viper config in the directory.
// Due to https://github.com/spf13/viper/issues/181,
// seems like there's not really a way to override with multiple config files.
// So we will read local config files into a separate viper instances, and then use MergeConfigMap with AllSettings.
func readLocalConfig(d string) {
	log.Debug.Printf("Reading config file in %s", d)

	v := viper.New()
	v.SetConfigName(".runtainer")
	v.AddConfigPath(d)
	if err := v.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.Debug.Printf("Local %s, skipping...", err)
		default:
			log.Error.Panic(err)
		}
	} else {
		log.Debug.Print("Using local config file:", v.ConfigFileUsed())
		if err := viper.MergeConfigMap(v.AllSettings()); err != nil {
			log.Error.Panic(err)
		}
	}
}

// splitArgs as per that POSIX standard, find the -- delimiter and split args by it
func splitArgs(args []string) ([]string, []string) {
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
