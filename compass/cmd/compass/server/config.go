package server

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/ossf/gemara/layer2"
	"github.com/ossf/gemara/layer4"

	"github.com/complytime/complybeacon/compass/mapper"
	"github.com/complytime/complybeacon/compass/mapper/factory"
)

func NewScopeFromCatalogPath(catalogPath string) (mapper.Scope, error) {
	cleanedPath := filepath.Clean(catalogPath)
	catalogData, err := os.ReadFile(cleanedPath)
	if err != nil {
		return nil, err
	}

	var layer2Catalog layer2.Catalog
	err = yaml.Unmarshal(catalogData, &layer2Catalog)
	if err != nil {
		return nil, err
	}

	return mapper.Scope{
		layer2Catalog.Metadata.Id: layer2Catalog,
	}, nil
}

type Config struct {
	Plugins     []PluginConfig `json:"plugins"`
	Certificate CertConfig     `json:"certConfig"`
}

type CertConfig struct {
	PublicKey  string `json:"cert"`
	PrivateKey string `json:"key"`
}

type PluginConfig struct {
	Id             string `json:"id"`
	EvaluationsDir string `json:"evaluations-dir"`
}

func NewMapperSet(config *Config) (mapper.Set, error) {
	pluginSet := make(mapper.Set)
	for _, pluginConf := range config.Plugins {
		transformerId := mapper.ID(pluginConf.Id)
		if pluginConf.EvaluationsDir == "" {
			log.Printf("Plugin %s has no evaluations, skipping...", transformerId)
			continue
		}

		info, err := os.Stat(pluginConf.EvaluationsDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return pluginSet, fmt.Errorf("evaluations directory %s for plugin %s: %w", pluginConf.EvaluationsDir, pluginConf.Id, err)
			}
			return pluginSet, err
		}

		if !info.IsDir() {
			return pluginSet, fmt.Errorf("evaluations directory %s for plugin %s is not a directory", pluginConf.EvaluationsDir, pluginConf.Id)
		}

		tfmr, err := NewMapperFromDir(transformerId, pluginConf.EvaluationsDir)
		if err != nil {
			return pluginSet, fmt.Errorf("unable to load configuration for %s: %w", pluginConf.Id, err)
		}
		pluginSet[transformerId] = tfmr
	}
	return pluginSet, nil
}

func NewMapperFromDir(pluginID mapper.ID, evaluationsPath string) (mapper.Mapper, error) {
	mpr := factory.MapperByID(pluginID)
	err := filepath.Walk(evaluationsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var evaluation []layer4.AssessmentPlan
		err = yaml.Unmarshal(content, &evaluation)
		if err != nil {
			return err
		}

		extension := filepath.Ext(info.Name())
		nameWithoutExt := strings.TrimSuffix(info.Name(), extension)

		// FIXME: Increase robustness here instead of relying on the filename.
		mpr.AddEvaluationPlan(nameWithoutExt, evaluation)
		return nil
	})
	return mpr, err
}
