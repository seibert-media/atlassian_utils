package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/seibert-media/atlassian-utils/confluence"
	command_list "github.com/bborbe/command/list"
	debian_config "github.com/seibert-media/debian-utils/config"
	debian_config_builder "github.com/seibert-media/debian-utils/config_builder"
	debian_config_parser "github.com/seibert-media/debian-utils/config_parser"
	debian_copier "github.com/seibert-media/debian-utils/copier"
	debian_package_creator "github.com/seibert-media/debian-utils/package_creator"
	debian_package_creator_archive "github.com/seibert-media/debian-utils/package_creator_archive"
	debian_package_creator_by_reader "github.com/seibert-media/debian-utils/package_creator_by_reader"
	debian_tar_gz_extractor "github.com/seibert-media/debian-utils/tar_gz_extractor"
	debian_zip_extractor "github.com/seibert-media/debian-utils/zip_extractor"
	http_client_builder "github.com/bborbe/http/client_builder"
	http_requestbuilder "github.com/bborbe/http/requestbuilder"
	"github.com/golang/glog"
	"strings"
)

const (
	PARAMETER_CONFIG            = "config"
	PARAMETER_TAR_GZ_PATH       = "path"
	PARAMETER_VERSION           = "version"
	PARAMETER_ATLASSIAN_VERSION = "atlassian-version"
	PARAMETER_TARGET            = "target"
)

type ConfigBuilderWithConfig func(config *debian_config.Config) debian_config_builder.ConfigBuilder

var (
	tarGzPathPtr        = flag.String(PARAMETER_TAR_GZ_PATH, "", "path to  tar gz")
	versionPtr          = flag.String(PARAMETER_VERSION, "", "version")
	atlassianVersionPtr = flag.String(PARAMETER_ATLASSIAN_VERSION, "", "atlassian version")
	configPtr           = flag.String(PARAMETER_CONFIG, "", "path to config")
	targetDirPtr        = flag.String(PARAMETER_TARGET, confluence.TARGET, "target")
)

func main() {
	defer glog.Flush()
	glog.CopyStandardLogTo("info")
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	commandListProvider := func() command_list.CommandList {
		return command_list.New()
	}
	config_parser := debian_config_parser.New()
	copier := debian_copier.New()
	zipExtractor := debian_zip_extractor.New()
	tarGzExtractor := debian_tar_gz_extractor.New()
	httpClientBuilder := http_client_builder.New().WithoutProxy()
	httpClient := httpClientBuilder.Build()
	requestbuilderProvider := http_requestbuilder.NewHTTPRequestBuilderProvider()
	debianPackageCreator := debian_package_creator.New(commandListProvider, copier, tarGzExtractor.ExtractTarGz, zipExtractor.ExtractZip, httpClient.Do, requestbuilderProvider.NewHTTPRequestBuilder)
	creatorByReader := debian_package_creator_by_reader.New(commandListProvider, debianPackageCreator, tarGzExtractor.ExtractTarGz)
	debianPackageCreatorArchive := debian_package_creator_archive.New(creatorByReader.CreatePackage)

	err := do(
		debianPackageCreatorArchive,
		config_parser,
		*tarGzPathPtr,
		*configPtr,
		*versionPtr,
		*atlassianVersionPtr,
		*targetDirPtr,
	)
	if err != nil {
		glog.Exit(err)
	}
}

func do(
	debianPackageCreatorArchive debian_package_creator_archive.DebianPackageCreator,
	config_parser debian_config_parser.ConfigParser,
	tarGzPath string,
	configpath string,
	version string,
	atlassianVersion string,
	targetDir string,
) error {
	if len(tarGzPath) == 0 {
		return fmt.Errorf("parameter %s missing", PARAMETER_TAR_GZ_PATH)
	}
	if len(atlassianVersion) == 0 {
		atlassianVersion = extractAtlassianVersion(version)
	}
	var err error
	config := createDefaultConfig()
	if len(configpath) > 0 {
		if config, err = config_parser.ParseFileToConfig(config, configpath); err != nil {
			return err
		}
	}
	config_builder := debian_config_builder.NewWithConfig(config)
	if len(version) > 0 {
		if err := config_builder.Version(version); err != nil {
			return err
		}
	}
	config = config_builder.Build()
	if len(config.Version) == 0 {
		return fmt.Errorf("parameter %s missing", PARAMETER_VERSION)
	}
	sourceDir := fmt.Sprintf("atlassian-confluence-%s", atlassianVersion)
	return debianPackageCreatorArchive.CreatePackage(tarGzPath, config, sourceDir, targetDir)
}

func extractAtlassianVersion(version string) string {
	pos := strings.IndexRune(version, '-')
	if pos == -1 {
		return version
	}
	return version[:pos]
}

func createDefaultConfig() *debian_config.Config {
	config := debian_config.DefaultConfig()
	config.Name = confluence.PACKAGE_NAME
	config.Architecture = confluence.ARCH
	return config
}
