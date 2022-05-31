package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

/*************************************/

// check if file exists (relative to the current directory)
func fileExists(path string) bool {

	// get current directory
	cwd, _ := os.Getwd()

	// get an absolute path of the file
	absPath := filepath.Join(cwd, path)

	// access file information
	if _, err := os.Stat(absPath); err != nil {
		return !os.IsNotExist(err) // return `false` if doesn't exist
	}

	// file exists
	return true
}

// copy file using `cp` command
func copyFile(src, dest string) {
	if err := exec.Command("cp", "-r", src, dest).Run(); err != nil {
		fmt.Println("An error occurred during copy operation:", src, "=>", dest)
		os.Exit(1)
	}
}

/*************************************/

// build the package for a platform
func build(packageName, destDir string, platform map[string]string, ldflags string, compress bool, extraFiles string) {

	// platform config
	platformKernel := platform["kernel"]
	platformArch := platform["arch"]

	// binary executable file path
	inputName := os.Getenv("INPUT_NAME")

	// build file name (same as the `inputName` if compression is enabled)
	buildFileName := fmt.Sprintf("%s-%s-%s", inputName, platformKernel, platformArch)
	if compress {
		buildFileName = inputName
	}

	// append `.exe` file-extension for windows
	if platformKernel == "windows" {
		buildFileName += ".exe"
	}

	// workspace directory
	workspaceDir := os.Getenv("GITHUB_WORKSPACE")

	// destination directory path
	destDirPath := filepath.Join(workspaceDir, destDir)

	// join destination path
	buildFilePath := filepath.Join(destDirPath, buildFileName)

	// package directory local path
	var packagePath string
	if packageName == "" {
		packagePath = "."
	} else {
		packagePath = "./" + packageName
	}

	// static build if building for same kernel and arch as this machine
	if runtime.GOOS == platformKernel && runtime.GOARCH == platformArch {
		if ldflags != "" {
			ldflags += " "
		}

		ldflags += "-linkmode external -extldflags \"-static\""
	}

	/*------------*/

	// command-line options for the `go build` command
	buildOptions := []string{"build", "-buildmode", "exe", "-ldflags", ldflags, "-o", buildFilePath, packagePath}

	// generate `go build` command
	buildCmd := exec.Command("go", buildOptions...)

	// set environment variables
	buildCmd.Env = append(os.Environ(), []string{
		fmt.Sprintf("GOOS=%s", platformKernel),
		fmt.Sprintf("GOARCH=%s", platformArch),
	}...)

	// execute `go build` command
	fmt.Println("Creating a build using :", buildCmd.String())
	if output, err := buildCmd.Output(); err != nil {
		fmt.Println("An error occurred during build:", err)
		os.Exit(1)
	} else {
		fmt.Printf("%s\n", output)
	}

	isWindows := platformKernel == "windows"

	/*------------------------------*/

	// create a compressed archive
	if compress {

		// compressed gzip file name
		outFileName := fmt.Sprintf("%s-%s-%s", inputName, platformKernel, platformArch)
		if isWindows {
			outFileName += ".zip"
		} else {
			outFileName += ".tar.gz"
		}

		/*------------*/

		// file to compress (default: build file)
		includeFiles := []string{buildFileName}

		// copy any extra files we want
		for _, f := range strings.Split(extraFiles, " ") {
			if fileExists(f) {
				copyFile(f, filepath.Join(destDirPath, f))
				includeFiles = append(includeFiles, f)
			}
		}

		/*------------*/

		// command-line options
		compressOption := "-cvzf"
		compressProgram := "tar"
		if isWindows {
			compressOption = "-9r"
			compressProgram = "zip"
		}

		compressOptions := append([]string{compressOption, outFileName}, includeFiles...)

		// generate the command
		cmd := exec.Command(compressProgram, compressOptions...)

		// set working directory for the command
		cmd.Dir = destDirPath

		// execute the command
		fmt.Println("Compressing build file using:", cmd.String())
		if err := cmd.Run(); err != nil {
			fmt.Println("An error occurred during compression:", err)
			os.Exit(1)
		}

		/*------------*/

		// generate cleanup command
		cleanCmd := exec.Command("rm", append([]string{"-rf"}, includeFiles...)...)

		// set working directory for the command
		cleanCmd.Dir = destDirPath

		// start cleanup process
		fmt.Println("Performing cleanup operation using:", cleanCmd.String())
		if err := cleanCmd.Run(); err != nil {
			fmt.Println("An error occurred during cleaup:", err)
			os.Exit(1)
		}

	}
}

/*************************************/

func main() {

	// get input variables from action
	inputPlatforms := os.Getenv("INPUT_PLATFORMS")
	inputPackage := os.Getenv("INPUT_PACKAGE")
	inputCompress := os.Getenv("INPUT_COMPRESS")
	inputDest := os.Getenv("INPUT_DEST")
	inputLdflags := os.Getenv("INPUT_LDFLAGS")
	extraFiles := os.Getenv("INPUT_EXTRAS")

	// package name to build
	packageName := strings.ReplaceAll(inputPackage, " ", "")

	// destination directory
	destDir := strings.ReplaceAll(inputDest, " ", "")

	// split platform names by comma (`,`)
	platforms := strings.Split(inputPlatforms, ",")

	// should compress build file
	compress := false
	if strings.ToLower(inputCompress) == "true" {
		compress = true
	}

	// for each platform, execute `build` function
	for _, platform := range platforms {

		// split platform by `/` (and clean all whitespaces)
		platformSpec := strings.Split(strings.ReplaceAll(platform, " ", ""), "/")

		// create a `map` of `kernel` and `arch`
		platformMap := map[string]string{
			"kernel": platformSpec[0],
			"arch":   platformSpec[1],
		}

		// execute `build` function
		build(packageName, destDir, platformMap, inputLdflags, compress, extraFiles)
	}

	/*------------*/

	// list files inside destination directory
	if output, err := exec.Command("ls", "-alh", destDir).Output(); err != nil {
		fmt.Println("An error occurred during ls operation:", err)
		os.Exit(1)
	} else {
		fmt.Println("--- BUILD FILES ---")
		fmt.Printf("%s\n", output)
	}

}
