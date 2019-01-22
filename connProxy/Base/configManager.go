// configManager
package base

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"os"
)

type ConfigManager struct {
	util     *Util
	Config   ProxyConfig
	FileName string
}

func NewConfigManager() (config *ConfigManager, err error) {

	config = new(ConfigManager)
	config.util = &Util{}
	tempPath, err := config.util.GetExecutePath()
	if err != nil {
		return nil, err
	}

	config.FileName = tempPath + `config.xml`
	//fmt.Println("ConfigPath=", config.fileName)

	return config, err
}

func (this *ConfigManager) LoadConfig() (config ProxyConfig, err error) {

	if ok, err := this.util.PathOrFileExists(this.FileName, 1); !ok {
		return config, err
	}
	//fmt.Println("load", 1)
	xmlBytes, err := ioutil.ReadFile(this.FileName)
	if err != nil {
		return config, err
	}
	//fmt.Println("load", 2, xmlBytes)
	err = xml.Unmarshal(xmlBytes, &this.Config)

	return this.Config, err
}

func (this *ConfigManager) SaveConfig(config *ProxyConfig) (err error) {
	if config != nil {

		tbytes, err := xml.MarshalIndent(config, "", "	")

		if err != nil {
			return err
		}

		err = ioutil.WriteFile(this.FileName, tbytes, os.ModePerm)

		if err != nil {
			return err
		}

		return nil
	}
	return errors.New("config is nil")
}
