package migrator

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CommitsHistory struct {
	History map[string]History
}

type History struct {
	GlobalID  string
	PathHash  string
	CommitID  string
	Timestamp time.Time
}

func MigrateToFakeCommitRepo(gitFolderOrigin string, gitFolderDestiny string) error {

	history, err := GetCommits(gitFolderOrigin)
	if err != nil {
		return err
	}

	for _, commit := range history.History {
		log.Printf("Timestamp: %s GlobalID: %s ", commit.Timestamp, commit.GlobalID)
	}

	err = CreateCommitsInRepo(gitFolderDestiny, history)
	if err != nil {
		return err
	}

	return nil

}

func CreateCommitsInRepo(gitFolder string, history *CommitsHistory) error {
	if gitFolder == "" {
		return errors.New("gitFolder path is empty")
	}

	log.Printf("Creating commits in %s", gitFolder)
	commitsExist, err := getCommitsFakeRepo(gitFolder)
	if err != nil {
		return err
	}

	log.Printf("Commits in %s: %d", gitFolder, len(commitsExist.History))

	for _, commit := range history.History {
		if _, ok := commitsExist.History[commit.GlobalID]; !ok {
			_, err := createCommitWithTimestampInRepo(gitFolder, commit.Timestamp, commit.GlobalID)
			if err != nil {
				return err
			}
		} else {
			log.Printf("[Skipping] Commit already exists in %s: %s", gitFolder, commit.GlobalID)
		}
	}

	return nil
}

func createCommitWithTimestampInRepo(gitFolder string, timestamp time.Time, globalId string) (string, error) {
	if _, err := os.Stat(gitFolder); os.IsNotExist(err) {
		return "", fmt.Errorf("gitFolder does not exist: %s", gitFolder)
	}

	command := fmt.Sprintf("cd %s && git commit --allow-empty -m \"%s\" --date \"%s\"", gitFolder, globalId, timestamp.Format(time.RFC3339))
	output, err := runCommand(command)
	if err != nil {
		return "", err
	}

	commitParts := strings.Fields(output)
	if len(commitParts) < 2 {
		return "", fmt.Errorf("unexpected output from git commit: %s", output)
	}

	commit := commitParts[1]
	return commit, nil
}

func GetCommits(gitFolder string) (*CommitsHistory, error) {
	if _, err := os.Stat(gitFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("gitFolder does not exist: %s", gitFolder)
	}

	log.Printf("Getting commits in %s", gitFolder)
	command := "cd " + gitFolder + " && git log --pretty=format:\"%H|%at\""
	output, err := runCommand(command)
	if err != nil {
		// Handle case where the branch does not have any commits yet
		if strings.Contains(err.Error(), "fatal: your current branch") && strings.Contains(err.Error(), "does not have any commits yet") {
			log.Printf("No commits found in branch: %s", gitFolder)
			return &CommitsHistory{History: make(map[string]History)}, nil
		}
		return nil, err
	}

	if output == "" {
		return &CommitsHistory{History: make(map[string]History)}, nil
	}

	hash := getHashFromString(gitFolder)
	history := make(map[string]History)
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		commit := strings.Split(line, "|")
		if len(commit) < 2 {
			log.Printf("invalid commit line: %s", line)
			continue
		}
		timestamp, err := strconv.ParseInt(commit[1], 10, 64)
		if err != nil {
			log.Printf("invalid timestamp for commit %s: %s", commit[0], commit[1])
			continue
		}
		globalId := fmt.Sprintf("%s-%s", hash, commit[0])
		history[globalId] = History{
			GlobalID:  globalId,
			PathHash:  hash,
			CommitID:  commit[0],
			Timestamp: time.Unix(timestamp, 0),
		}
	}

	return &CommitsHistory{History: history}, nil
}

func getCommitsFakeRepo(gitFolder string) (*CommitsHistory, error) {
	if _, err := os.Stat(gitFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("gitFolder does not exist: %s", gitFolder)
	}

	log.Printf("Getting commits in %s", gitFolder)
	command := "cd " + gitFolder + " && git log --pretty=format:\"%H|%at|%s\""
	output, err := runCommand(command)
	if err != nil {
		// Handle case where the branch does not have any commits yet
		if strings.Contains(err.Error(), "fatal: your current branch") && strings.Contains(err.Error(), "does not have any commits yet") {
			log.Printf("No commits found in branch: %s", gitFolder)
			return &CommitsHistory{History: make(map[string]History)}, nil
		}
		return nil, err
	}

	if output == "" {
		return &CommitsHistory{History: make(map[string]History)}, nil
	}

	history := make(map[string]History)
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		commit := strings.Split(line, "|")
		if len(commit) < 3 {
			log.Printf("invalid commit line: %s", line)
			continue
		}
		timestamp, err := strconv.ParseInt(commit[1], 10, 64)
		if err != nil {
			log.Printf("invalid timestamp for commit %s: %s", commit[0], commit[1])
			continue
		}
		hashRepo := strings.Split(gitFolder, "-")
		hash := hashRepo[0]

		globalId := commit[2]
		history[globalId] = History{
			GlobalID:  globalId,
			PathHash:  hash,
			CommitID:  commit[0],
			Timestamp: time.Unix(timestamp, 0),
		}
	}

	return &CommitsHistory{History: history}, nil
}

func getHashFromString(s string) string {
	hash := fnv.New32a()
	hash.Write([]byte(s))
	return strconv.Itoa(int(hash.Sum32()))
}

func runCommand(command string) (string, error) {
	log.Printf("Running command: %s", command)
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s, output: %s", err, string(output))
	}
	return string(output), nil
}
