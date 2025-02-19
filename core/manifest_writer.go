package core

import (
	"bufio"
	"fmt"
	"os"
	"sort"
)

func (mg *ManifestGenerator) WriteManifest(manifest Manifest, manifestFile string) error {
	file, err := os.Create(manifestFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString("filelist:\n")
	if err != nil {
		return err
	}

	var sortedFiles []string
	for file := range manifest.FileList {
		sortedFiles = append(sortedFiles, file)
	}
	sort.Strings(sortedFiles)

	for _, file := range sortedFiles {
		isCommented := manifest.FileList[file]
		prefix := "- "
		if isCommented {
			prefix = "# - "
		}
		_, err = writer.WriteString(fmt.Sprintf("%s%s\n", prefix, file))
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}
