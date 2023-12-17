package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:   "Go-Week",
		Usage:  "generate your weekly report with one click",
		Action: mainAction,
		Authors: []*cli.Author{
			{
				Name:  "Edward Chen",
				Email: "hi@edch.top",
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "git",
				Aliases: []string{"g"},
				Usage:   "git add, commit and push the weekly report",
			},
			&cli.BoolFlag{
				Name:    "last-week",
				Aliases: []string{"l"},
				Usage:   "generate the weekly report of last week",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func mainAction(ctx *cli.Context) error {
	commit := ctx.Bool("git")
	lastWeek := ctx.Bool("last-week")
	if commit {
		err := gitCommit()
		if err != nil {
			return err
		}
		return nil
	}

	template, err := readTemplate()
	if err != nil {
		return nil
	}

	content, err := fillTemplate(template, lastWeek)
	if err != nil {
		return err
	}

	file, err := writeToFile(content, lastWeek)
	if err != nil {
		return err
	}

	return openFileWithTypora(file)
}

var profileDir = filepath.Join(os.Getenv("USERPROFILE"), ".goweek")

func readTemplate() (string, error) {
	templatePath := filepath.Join(profileDir, "template.md")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func gitCommit() error {
	goWeekConfig, err := readConfig()
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = goWeekConfig.DocsDir
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	if len(out) == 0 {
		fmt.Println("No changes to commit")
		return nil
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = goWeekConfig.DocsDir
	err = cmd.Run()
	if err != nil {
		return err
	}
	fmt.Println("Add files to git...")

	cmd = exec.Command("git", "commit", "-m", "update weekly report")
	cmd.Dir = goWeekConfig.DocsDir
	err = cmd.Run()
	if err != nil {
		return err
	}
	fmt.Println("Commit files to git...")

	cmd = exec.Command("git", "push")
	cmd.Dir = goWeekConfig.DocsDir
	err = cmd.Run()
	if err != nil {
		return err
	}
	fmt.Println("Push files to git!")

	return nil
}

type DateInfo struct {
	Week      string
	WeekStart string
	WeekEnd   string
	Month     string
}

func getDateInfo(lastWeek bool) (DateInfo, error) {
	loc, err := time.LoadLocation("Asia/Hong_Kong")
	if err != nil {
		return DateInfo{}, err
	}

	var dateInfo DateInfo

	now := time.Now().In(loc)
	if lastWeek {
		now = now.AddDate(0, 0, -7)
	}

	_, weekNo := now.ISOWeek()
	dateInfo.Week = fmt.Sprintf("%02d", weekNo)

	weekDay := int(now.Weekday())
	if now.Weekday() == time.Sunday {
		weekDay = 7
	}

	weekStart := now.AddDate(0, 0, -weekDay+1)
	dateInfo.WeekStart = weekStart.Format("2006/01/02")

	weekEnd := now.AddDate(0, 0, 5-weekDay)
	dateInfo.WeekEnd = weekEnd.Format("2006/01/02")

	dateInfo.Month = now.Format("2006-01")

	return dateInfo, nil
}

func fillTemplate(tpl string, lastWeek bool) (string, error) {
	dateInfo, err := getDateInfo(lastWeek)
	if err != nil {
		return "", err
	}

	// Replace the placeholders
	tpl = strings.ReplaceAll(tpl, "{{.Week}}", dateInfo.Week)
	tpl = strings.ReplaceAll(tpl, "{{.WeekStart}}", dateInfo.WeekStart)
	tpl = strings.ReplaceAll(tpl, "{{.WeekEnd}}", dateInfo.WeekEnd)
	return tpl, nil
}

type GoWeekConfig struct {
	DocsDir    string `json:"docs_dir"`
	TyporaPath string `json:"typora_path"`
}

func readConfig() (GoWeekConfig, error) {
	configPath := filepath.Join(profileDir, "config.json")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return GoWeekConfig{}, err
	}

	var goWeekConfig GoWeekConfig
	err = json.Unmarshal(content, &goWeekConfig)
	if err != nil {
		return GoWeekConfig{}, err
	}

	return goWeekConfig, nil
}

func writeToFile(content string, lastWeek bool) (string, error) {
	goWeekConfig, err := readConfig()
	if err != nil {
		return "", err
	}

	dateInfo, err := getDateInfo(lastWeek)
	if err != nil {
		return "", err
	}

	monthDir := filepath.Join(goWeekConfig.DocsDir, dateInfo.Month)
	_, err = os.Stat(monthDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(monthDir, 0755)
		if err != nil {
			return "", err
		}
	}

	file := filepath.Join(monthDir, fmt.Sprintf("weekly-report-%s.md", dateInfo.Week))
	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		err := os.WriteFile(file, []byte(content), 0644)
		if err != nil {
			return "", err
		}
	} else {
		fmt.Printf("file \"%s\" already exists", file)
		return file, nil
	}

	fmt.Printf("Weekly report saved to \"%s\" successfully", file)
	return file, nil
}

func openFileWithTypora(file string) error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	cmd := exec.Command(config.TyporaPath, "--fullscreen", file)
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
