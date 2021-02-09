package cli

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
)

type Cli struct {
	NodeName     string
	Role         string
	UploadMethod string
	CaseNumber   string
	Obfuscate    bool
	NfsShare     string
	NfsOptions   string
	FtpServer    string
	Username     string
	Password     string
	LogLevel     log.Level
	DryRun       bool
	YamlDir      string
}

func NewCli() (*Cli, error) {
	c := new(Cli)

	nodeName := flag.String("node", "", "Run Sosreport on this node only")
	role := flag.String("role", "", "Run Sosreport on this role only")
	uploadMethod := flag.String("upload-method", "none", "Specify an upload method <none|case|ftp|nfs>")
	caseNumber := flag.String("case-number", "", "Specify a case number for an RH case to upload to")
	obfuscate := flag.Bool("obfuscate", false, "Obfuscate sosreport credentials")
	nfsShare := flag.String("nfs-share", "", "Specify an NFS share to upload to")
	nfsOptions := flag.String("nfs-options", "", "Specify an NFS options")
	ftpServer := flag.String("ftp-server", "", "Specify an FTP server to upload to")
	username := flag.String("username", "", "Username for RH portal or FTP")
	password := flag.String("password", "", "Password for RH portal or FTP")
	logLevel := flag.String("log-level", "", "Log level for this application")
	dryRun := flag.Bool("dry-run", false, "Dry run - generate YAML only")
	yamlDir := flag.String("yaml-dir", "", "If this is specified, write YAML files to dir (local dir otherwise)")

	flag.Parse()

	c.NodeName = *nodeName
	c.Role = *role
	c.UploadMethod = *uploadMethod
	c.CaseNumber = *caseNumber
	c.Obfuscate = *obfuscate
	c.NfsShare = *nfsShare
	c.NfsOptions = *nfsOptions
	c.FtpServer = *ftpServer
	c.Username = *username
	c.Password = *password
	switch *logLevel {
	case "panic":
		c.LogLevel = log.PanicLevel
	case "fatal":
		c.LogLevel = log.FatalLevel
	case "error":
		c.LogLevel = log.ErrorLevel
	case "warn":
		c.LogLevel = log.WarnLevel
	case "info":
		c.LogLevel = log.InfoLevel
	case "debug":
		c.LogLevel = log.DebugLevel
	case "trace":
		c.LogLevel = log.TraceLevel
	default:
		c.LogLevel = log.InfoLevel
	}
	c.DryRun = *dryRun
	c.YamlDir = *yamlDir

	return c, nil
}

func (c *Cli) PrintFlags() string {
	return fmt.Sprintf("%s: %v\n%s: %v\n%s: %v\n%s: %v\n%s: %v\n%s: %v\n%s: %v\n%s: %v\n%s: %v\n%s: %v\n%s: %v",
		"NodeName", c.NodeName,
		"Role", c.Role,
		"UploadMethod", c.UploadMethod,
		"CaseNumber", c.CaseNumber,
		"Obfuscate", c.Obfuscate,
		"NfsShare", c.NfsShare,
		"NfsOptions", c.NfsOptions,
		"FtpServer", c.FtpServer,
		"Username", c.Username,
		"Password", c.Password,
		"LogLevel", c.LogLevel,
	)
}
