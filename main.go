package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"text/template"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	configFile := flag.String("f", "docker-volume.yaml", "config file")
	debugLevel := flag.Bool("d", false, "enable debug logging")
	flag.Parse()

	if *debugLevel {
		log.SetLevel(log.DebugLevel)
	}

	viper.SetConfigFile(*configFile)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("fatal error config file: %w", err)
	}

	p := &volumePlugin{}
	h := volume.NewHandler(p)

	err = viper.Unmarshal(&p.config, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		stringToTemplateDecodeHookFunc(),
	)))
	if err != nil {
		log.Fatalf("unable to decode config into structs, %v", err)
	}

	if err := os.RemoveAll(p.config.Socket); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", p.config.Socket)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	log.Infof("volumePlugin starting on " + p.config.Socket)
	log.Debugf("Config: %+v", p.config)

	// p.Create(&volume.CreateRequest{
	// 	Name: "rbddemo4_hello-rbd",
	// })
	h.Serve(l)
}

func stringToTemplateDecodeHookFunc() mapstructure.DecodeHookFuncType {
	return func(from, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}

		if to != reflect.TypeOf(template.Template{}) {
			return data, nil
		}

		return template.New("templ").
			Parse(data.(string))
	}
}

type volumePlugin struct {
	config config
}

type config struct {
	Socket    string
	Defaults  map[string]string
	Templates TemplatesConfig
}

type TemplatesConfig struct {
	Create  template.Template
	Mount   template.Template
	Unmount template.Template
	Remove  template.Template
	Get     template.Template
	Path    template.Template
	List    template.Template
}

func templatedCmd(logCtx *log.Entry, tmplCmd template.Template, templateFill map[string]string) ([]string, error) {
	// Fill out the template to get the shell script out
	var shellCommands bytes.Buffer
	err := tmplCmd.Execute(&shellCommands, templateFill)
	if err != nil {
		return nil, err
	}

	// Special processing for STDOUT and STDERR
	// 1. Need to collect _just_ STDOUT, break it into lines for successful return result
	// 2. Need to collect _both_ STDOUT and STDERR and print it 'live' to the log (e.g. shell partially executes and hangs)
	// 3. Need to collect _both_ STDOUT and STDERR and return it in the error message

	cmd := exec.Command("/bin/bash", "-cex", shellCommands.String())
	stdoutOrig, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	var stdOutBuffer bytes.Buffer
	stdout := io.TeeReader(stdoutOrig, &stdOutBuffer)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var allLines []string
	in := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for in.Scan() {
		lastLine := in.Text()
		allLines = append(allLines, lastLine)
		logCtx.Debug(lastLine)
	}
	if err := in.Err(); err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			logCtx.Warnf("Command failed (CASE1), %s", exiterr)
		} else {
			logCtx.Warnf("Command failed (CASE2), %s", err)
		}
		return nil, fmt.Errorf("%s", strings.Join(allLines, "\n"))
	}

	var stdOutLines []string
	in = bufio.NewScanner(&stdOutBuffer)
	for in.Scan() {
		stdOutLines = append(stdOutLines, in.Text())
	}
	if err := in.Err(); err != nil {
		return nil, err
	}
	return stdOutLines, nil
}
