/*
Copyright © 2020 Stamus Networks oss@stamus-networks.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"gopherCap/pkg/fs"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// tarExtractCmd represents the tarExtract command
var tarExtractCmd = &cobra.Command{
	Use:   "tarExtract",
	Short: "Extract selected pcap files from tar.gz",
	Long: `Got a huge tar.gz file with pcaps with no space to unpack it? Fear not, just iterate over the bytestream and unpack only the files you need. Directly to gzip if needed.

Example usage:
gopherCap tarExtract \
	--in-tarball /mnt/ext/tarball.tar.gz \
	--file-regexp "pcap-2020.+\.pcap" \
	--out-dir /mnt/nfs/ \
	--out-gzip

List but don't extract anything:
gopherCap tarExtract \
	--in-tarball /mnt/ext/tarball.tar.gz \
	--file-regexp "pcap-2020.+\.pcap" \
	--dryrun
`,
	Run: func(cmd *cobra.Command, args []string) {
		fileReader, err := fs.Open(viper.GetString("in.tarball"))
		if err != nil {
			logrus.Fatalf("Tarball read: %s", err)
		}
		defer fileReader.Close()
		reader := tar.NewReader(fileReader)
		pattern := func() *regexp.Regexp {
			if pattern := viper.GetString("file.regexp"); pattern != "" {
				re, err := regexp.Compile(pattern)
				if err != nil {
					logrus.Fatal(err)
				}
				return re
			}
			return nil
		}()
		outDir := viper.GetString("out.dir")
		if outDir == "" && !viper.GetBool("dryrun") {
			logrus.Fatal("Missing output dir.")
		}
		logrus.Infof("Starting reater for %s.", viper.GetString("in.tarball"))
	loop:
		for {
			header, err := reader.Next()
			if err == io.EOF {
				logrus.Debug("EOF, stopping reader.")
				break loop
			}
			if err != nil {
				logrus.Fatalf("Tarball read error: %s", err)
			}
			logrus.Debugf("Found %s", header.Name)
			info := header.FileInfo()
			if info.IsDir() {
				logrus.Debugf("%s is a folder, skipping", info.Name())
				continue loop
			}
			if pattern == nil || (pattern != nil && pattern.MatchString(header.Name)) {
				logrus.Infof("%s matches file pattern", header.Name)
				outfile := filepath.Join(
					outDir, strings.ReplaceAll(strings.TrimPrefix(info.Name(), "./"), "/", "-"),
				)
				if viper.GetBool("out.gzip") {
					outfile = fmt.Sprintf("%s.gz", outfile)
				}
				file, err := os.OpenFile(outfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
				if err != nil {
					logrus.Fatal(err)
				}
				var writer io.WriteCloser = file
				if viper.GetBool("out.gzip") {
					writer = gzip.NewWriter(file)
				}
				_, err = io.Copy(writer, reader)
				if err != nil {
					logrus.Error(err)
				}
				writer.Close()
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(tarExtractCmd)

	tarExtractCmd.PersistentFlags().String("in-tarball", "",
		`Input gzipped tarball.`)
	viper.BindPFlag("in.tarball", tarExtractCmd.PersistentFlags().Lookup("in-tarball"))

	tarExtractCmd.PersistentFlags().String("out-dir", "",
		`Output directory for pcap files.`)
	viper.BindPFlag("out.dir", tarExtractCmd.PersistentFlags().Lookup("out-dir"))

	tarExtractCmd.PersistentFlags().Bool("dryrun", false,
		`Only list files in tarball for regex validation, do not extract.`)
	viper.BindPFlag("dryrun", tarExtractCmd.PersistentFlags().Lookup("dryrun"))

	tarExtractCmd.PersistentFlags().Bool("out-gzip", false,
		`Compress extracted files with gzip.`)
	viper.BindPFlag("out.gzip", tarExtractCmd.PersistentFlags().Lookup("out-gzip"))
}
