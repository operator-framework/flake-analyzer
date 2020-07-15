package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	owner string = "operator-framework"
	repo  string = "Operator-lifecycle-manager"
)

func TestMain(m *testing.M) {
	err := os.Chdir("../..")
	if err != nil {
		fmt.Printf("could not change dir: %v", err)
		os.Exit(1)
	}

	fmt.Println("building binary...")
	build := exec.Command("make")
	err = build.Run()
	if err != nil {
		fmt.Printf("could not build binary: %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestPeriodicAnalysis(t *testing.T) {
	err := os.Chdir("../..")
	if err != nil {
		fmt.Printf("could not change dir: %v", err)
		os.Exit(1)
	}

	token := os.Getenv("GITHUB_TOKEN")

	tests := []struct {
		name string
		args []string
	}{
		{name: "Generate report by time frame",
			args: []string{"--from=7", "--to=0", "-f=e2e-test-output", "-o=./recentreport"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd:= exec.Command("./bin/flake-analyzer", append([]string{"-n=" + owner, "-r=" + repo, "-t=" + token},
			tt.args...)...)

			output,err:=cmd.CombinedOutput()
			assert.NoError(t,err)
			assert.NotEmpty(t,output)
			fmt.Println(output)
		})
	}

}
