package launcher

import (
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
)

var errCancelled = errors.New("cancelled by user")

// Launcher is a JNLP Launcher
type Launcher interface {
	SetWorkDir(dir string)
	SetWindowTitle(title string)
	RunByFilename(filename string) error
	RunByURL(url string) error
	SetOptions(options *Options)
	Terminate()
	CheckPlatform() error
	UninstallByFilename(filename string, showGUI bool) error
	UninstallByURL(url string, showGUI bool) error
	SetLogFile(logFile string)
	Wait() (*os.ProcessState, error)
}

// OutputHandler is invoked with the read end of a pipe to which the Launcher forwards
// output (stdout or stderr). Each instance of an OutputHandler will be provided its own
// goroutine, so it may block as necessary.
type OutputHandler = func(pipe io.ReadCloser)

type Options struct {
	IsRunningFromBrowser          bool
	JavaDir                       string
	ShowConsole                   bool
	DisableVerification           bool
	DisableVerificationSameOrigin bool

	// If non-nil, processes output from stdout of the launched process
	StdoutHandler OutputHandler
	// If non-nil, processes output from stderr of the launched process
	StderrHandler OutputHandler
}

func RegisterProtocol(scheme string, launcher Launcher) {
	protocolLaunchers[scheme] = launcher
}

func RegisterExtension(ext string, launcher Launcher) {
	extensionLaunchers[ext] = launcher
}

func FindLauncherForURL(rawurl string) (Launcher, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if !parsedURL.IsAbs() {
		return nil, errors.Errorf("URL %s is not absolute", rawurl)
	}
	scheme := parsedURL.Scheme
	if launcher, ok := protocolLaunchers[scheme]; ok {
		return launcher, nil
	}
	if scheme == "http" || scheme == "https" {
		for ext, launcher := range extensionLaunchers {
			if strings.HasSuffix(rawurl, "."+ext) {
				return launcher, nil
			}
		}
	}
	return nil, errors.Errorf("unable to find launcher for URL %s", rawurl)
}

func FindLauncherForExtension(path string) (Launcher, error) {
	for ext, launcher := range extensionLaunchers {
		if strings.HasSuffix(path, "."+ext) {
			return launcher, nil
		}
	}
	return nil, errors.Errorf("unable to find launcher for path %s", path)
}

func FindLauncherForURLOrFilename(filenameOrURL string) (launcher Launcher, byURL bool, err error) {
	var myLauncher Launcher
	var errURL, errExt error
	byURL = false
	myLauncher, errURL = FindLauncherForURL(filenameOrURL)
	if errURL != nil {
		myLauncher, errExt = FindLauncherForExtension(filenameOrURL)
		if errExt != nil {
			return nil, false, errors.Errorf("unable to handle filename or URL %s: (%v, %v)", filenameOrURL, errURL, errExt)
		}
	} else {
		byURL = true
	}
	return myLauncher, byURL, nil
}

var protocolLaunchers = make(map[string]Launcher)
var extensionLaunchers = make(map[string]Launcher)
