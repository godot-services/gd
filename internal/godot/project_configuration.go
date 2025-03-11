package godot

import (
	"fmt"

	"github.com/zieckey/goini"
)

type projectConfiguration struct {
	targetFilePath string

	PackageTitle   *string
	PackageVersion *string
	PackageLicense *string
	PackagePrivate bool

	EditorVersion string

	Assets    []projectAssetConfiguration
	DevAssets []projectAssetConfiguration
}

type projectAssetConfiguration struct{}

func NewProjectConfiguration(targetFilePath string) (*projectConfiguration, error) {
	ini := goini.New()
	err := ini.ParseFile(targetFilePath)
	if err != nil {
		fmt.Printf("parse INI file %v failed : %v\n", targetFilePath, err.Error())
		return nil, err
	}

	fmt.Println(ini.GetAll())

	config := &projectConfiguration{
		targetFilePath: targetFilePath,
	}

	return config, nil
}
