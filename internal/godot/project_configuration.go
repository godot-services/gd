package godot

import "github.com/spf13/viper"

type projectConfiguration struct {
	configViper    *viper.Viper
	targetFilePath string

	PackageLicense *string `mapstructure:"package.license"`
	PackagePrivate bool    `mapstructure:"package.private"`
	PackageTitle   *string `mapstructure:"package.title"`
	PackageVersion *string `mapstructure:"package.version"`

	EditorVersion string

	Assets    []projectAssetConfiguration
	DevAssets []projectAssetConfiguration
}

type projectAssetConfiguration struct{}

func NewProjectConfiguration(targetFilePath string) (*projectConfiguration, error) {
	projectConfViper := viper.New()
	projectConfViper.SetConfigFile(targetFilePath)
	projectConfViper.SetConfigType("ini")
	err := projectConfViper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	config := &projectConfiguration{
		configViper:    projectConfViper,
		targetFilePath: targetFilePath,
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
