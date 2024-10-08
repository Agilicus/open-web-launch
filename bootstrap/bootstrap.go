package bootstrap

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/rocketsoftware/open-web-launch/launcher"
	"github.com/rocketsoftware/open-web-launch/messaging"
	"github.com/rocketsoftware/open-web-launch/settings"
	"github.com/rocketsoftware/open-web-launch/utils"
	"github.com/rocketsoftware/open-web-launch/utils/log"
)

var (
	javaDir                       string
	showConsole                   bool
	disableVerification           bool
	disableVerificationSameOrigin bool
	uninstall                     bool
	showGUI                       bool
)

var helpOptions = []string{"-help", "--help", "/help", "-?", "/?"}

func Run(productName, productTitle, productVersion string) {
	usage := func() { showUsage(productTitle, productVersion); os.Exit(2) }
	if len(os.Args) == 1 {
		usage()
	}
	if len(os.Args) > 1 {
		for _, helpOption := range helpOptions {
			if helpOption == os.Args[1] {
				usage()
			}
		}
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	userConfigDir, err := os.UserConfigDir()

	userConfigDir = filepath.Join(userConfigDir, "Rocket Software")
	if err := utils.CreateProductWorkDir(userConfigDir); err != nil {
		log.Fatal(err)
	}
	userConfigDir = filepath.Join(userConfigDir, productTitle)
	if err := utils.CreateProductWorkDir(userConfigDir); err != nil {
		log.Fatal(err)
	}

	productWorkDir := filepath.Join(userConfigDir, "cache")
	if err := utils.CreateProductWorkDir(productWorkDir); err != nil {
		log.Fatal(err)
	}

	productLogFile := filepath.Join(userConfigDir, "log")
	if err := utils.CreateProductWorkDir(productLogFile); err != nil {
		log.Fatal(err)
	}
	productLogFile = filepath.Join(productLogFile, productName+".log")

	fmt.Fprintf(os.Stderr, "%s %s\n", productTitle, productVersion)
	logFile, err := utils.OpenOrCreateProductLogFile(productLogFile)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logFile)
	log.Printf("starting %s %s with arguments %v\n", productTitle, productVersion, os.Args)
	log.Printf("current platform is OS=%q Architecture=%q\n", runtime.GOOS, runtime.GOARCH)
	log.Printf("proxy settings are HTTP_PROXY=%s HTTPS_PROXY=%s NO_PROXY=%s UseHttpProxyEnvironmentVariable is %v\n",
		os.Getenv("HTTP_PROXY"), os.Getenv("HTTPS_PROXY"), os.Getenv("NO_PROXY"),
		settings.UseHttpProxyEnvironmentVariable(),
	)
	if !settings.UseHttpProxyEnvironmentVariable() {
		os.Setenv("HTTP_PROXY", "")
		os.Setenv("HTTPS_PROXY", "")
		os.Setenv("NO_PROXY", "")
	}
	flag.BoolVar(&showConsole, "showconsole", false, "show Java console")
	flag.BoolVar(&showConsole, "showConsole", false, "show Java console")
	flag.StringVar(&javaDir, "javadir", "", "Java folder that should be used for starting a Java Web Start application")
	flag.StringVar(&javaDir, "javaDir", "", "Java folder that should be used for starting a Java Web Start application")
	flag.BoolVar(&disableVerification, "disableverification", false, "don't verify jar signatures")
	flag.BoolVar(&disableVerification, "disableVerification", false, "don't verify jar signatures")
	flag.BoolVar(&disableVerificationSameOrigin, "disableverificationsameorigin", false, "don't verify all jars have same signature")
	flag.BoolVar(&disableVerificationSameOrigin, "disableVerificationSameOrigin", false, "don't verify all jars have same signature")
	flag.BoolVar(&uninstall, "uninstall", false, "uninstall a specific Java Web Start application")
	flag.BoolVar(&showGUI, "gui", false, "show GUI")
	flag.Usage = usage
	flag.Parse()
	argCount := flag.NArg()
	flagCount := flag.NFlag()
	if argCount == 1 && flagCount == 0 && !strings.HasPrefix(flag.Arg(0), "chrome-extension://") {
		filenameOrURL := flag.Arg(0)
		handleURLOrFilename(filenameOrURL, nil, productWorkDir, productTitle, productLogFile)
	} else if argCount == 1 && uninstall {
		filenameOrURL := flag.Arg(0)
		handleUninstallCommand(filenameOrURL, showGUI, productWorkDir, productTitle, productLogFile)
	} else if argCount == 1 && !strings.HasPrefix(flag.Arg(0), "chrome-extension://") {
		filenameOrURL := flag.Arg(0)
		options := &launcher.Options{}
		if isFlagSet("javadir") || isFlagSet("javaDir") {
			var err error
			if javaDir, err = settings.UseJavaDir(javaDir); err != nil {
				log.Fatal(err)
			}
			options.JavaDir = javaDir
		}
		if isFlagSet("showconsole") || isFlagSet("showConsole") {
			settings.ShowConsole()
			options.ShowConsole = true
		}
		if isFlagSet("disableverification") || isFlagSet("disableVerification") {
			settings.DisableVerification()
			options.DisableVerification = true
		}
		if isFlagSet("disableverificationsameorigin") || isFlagSet("disableVerificationSameOrigin") {
			settings.DisableVerificationSameOrigin()
			options.DisableVerificationSameOrigin = true
		}
		handleURLOrFilename(filenameOrURL, options, productWorkDir, productTitle, productLogFile)
	} else {
		isRunningFromBrowser := true
		options := &launcher.Options{IsRunningFromBrowser: isRunningFromBrowser}
		log.Printf("running from browser: %v", isRunningFromBrowser)
		listenForMessage(options, productWorkDir, productTitle, productLogFile)
	}
}

func handleURLOrFilename(filenameOrURL string, options *launcher.Options, productWorkDir string, productTitle string, productLogFile string) {
	myLauncher, byURL, err := launcher.FindLauncherForURLOrFilename(filenameOrURL)
	if err != nil {
		log.Fatal(err)
	}
	myLauncher.SetLogFile(productLogFile)
	myLauncher.SetWorkDir(productWorkDir)
	myLauncher.SetWindowTitle(productTitle)
	myLauncher.SetOptions(options)
	if byURL {
		if err := myLauncher.RunByURL(filenameOrURL); err != nil {
			log.Fatal(err)
			return
		}
	} else {
		if err := myLauncher.RunByFilename(filenameOrURL); err != nil {
			log.Fatal(err)
			return
		}
	}
}

func listenForMessage(options *launcher.Options, productWorkDir string, productTitle string, productLogFile string) {
	message, err := messaging.GetMessage(os.Stdin)
	if err != nil {
		if errors.Cause(err) != io.EOF {
			log.Fatal(err)
		}
		log.Println("exit because stdin has been closed")
		return
	}
	if message.Status != "" {
		response := fmt.Sprintf(`{"status": "installed"}`)
		if err := messaging.SendMessage(os.Stdout, response); err != nil {
			log.Fatal(err)
		}
		return
	}
	myLauncher, err := launcher.FindLauncherForURL(message.URL)
	if err != nil {
		log.Fatal(err)
	}
	myLauncher.SetLogFile(productLogFile)
	myLauncher.SetWorkDir(productWorkDir)
	myLauncher.SetWindowTitle(productTitle)
	myLauncher.SetOptions(options)
	if err := myLauncher.RunByURL(message.URL); err != nil {
		stringError := fmt.Sprintf("%v", err)
		jsonError, _ := json.Marshal(stringError)
		response := fmt.Sprintf(`{"status": %s}`, string(jsonError))
		log.Println(response)
		if err := messaging.SendMessage(os.Stdout, response); err != nil {
			log.Fatal(err)
		}
		return
	}
	response := fmt.Sprintf(`{"status": "ok"}`)
	if err := messaging.SendMessage(os.Stdout, response); err != nil {
		log.Fatal(err)
	}
}

func handleUninstallCommand(filenameOrURL string, showGUI bool, productWorkDir string, productTitle string, productLogFile string) {
	myLauncher, byURL, err := launcher.FindLauncherForURLOrFilename(filenameOrURL)
	if err != nil {
		log.Fatal(err)
	}
	myLauncher.SetLogFile(productLogFile)
	myLauncher.SetWorkDir(productWorkDir)
	myLauncher.SetWindowTitle(productTitle)
	if byURL {
		if err := myLauncher.UninstallByURL(filenameOrURL, showGUI); err != nil {
			log.Fatal(err)
			return
		}
	} else {
		if err := myLauncher.UninstallByFilename(filenameOrURL, showGUI); err != nil {
			log.Fatal(err)
			return
		}
	}
}

func isFlagSet(flagName string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			found = true
		}
	})
	return found
}

func buildUsageText(productTitle, productVersion string) string {
	program := filepath.Base(os.Args[0])
	var text string
	text += fmt.Sprintf("%s %s\n", productTitle, productVersion)
	text += fmt.Sprintf("\n")
	text += fmt.Sprintf("Usage:\n")
	text += fmt.Sprintf("%s [options] <filename | URL>\n", program)
	text += fmt.Sprintf("\n")
	text += fmt.Sprintf("Options:\n")
	text += fmt.Sprintf("  -javaDir <java folder>\n")
	text += fmt.Sprintf("      use Java from <java folder>\n")
	text += fmt.Sprintf("  -showConsole\n")
	text += fmt.Sprintf("      show Java console\n")
	text += fmt.Sprintf("  -disableVerification\n")
	text += fmt.Sprintf("      don't verify jar signatures\n")
	text += fmt.Sprintf("  -disableVerificationSameOrigin\n")
	text += fmt.Sprintf("      don't verify all jars have same signature\n")
	text += fmt.Sprintf("  -uninstall\n")
	text += fmt.Sprintf("      uninstall app\n")
	text += fmt.Sprintf("  -gui\n")
	text += fmt.Sprintf("      show GUI, uninstall only\n")
	text += fmt.Sprintf("  -help\n")
	text += fmt.Sprintf("      show help\n")
	return text
}

func showUsage(productTitle, productVersion string) {
	text := buildUsageText(productTitle, productVersion)
	utils.ShowUsage(productTitle, productVersion, text)
}
