package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var (
	longFormat bool
	showHidden bool
	sortField  string
	showIcons  bool
	minSize    string
	maxSize    string
	dirsFirst  bool
)

type Config struct {
	LongFormat bool   `json:"long"`
	ShowHidden bool   `json:"all"`
	SortField  string `json:"sort"`
	ShowIcons  bool   `json:"icons"`
	MinSize    string `json:"min_size"`
	MaxSize    string `json:"max_size"`
	DirsFirst  bool   `json:"dirs_first"`
}

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

func main() {
	config := loadConfig()

	flag.BoolVar(&longFormat, "l", config.LongFormat, "use a long listing format")
	flag.BoolVar(&showHidden, "a", config.ShowHidden, "do not ignore entries starting with .")
	flag.StringVar(&sortField, "sort", config.SortField, "sort by field: name, size, time, ext")
	if config.SortField == "" {
		sortField = "name"
	}
	flag.BoolVar(&showIcons, "icons", config.ShowIcons, "show icons")
	flag.StringVar(&minSize, "min-size", config.MinSize, "filter by minimum size (e.g. 10K, 1M)")
	flag.StringVar(&maxSize, "max-size", config.MaxSize, "filter by maximum size (e.g. 10K, 1M)")
	flag.BoolVar(&dirsFirst, "dirs-first", config.DirsFirst, "list directories first")
	flag.Parse()

	args := flag.Args()
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, path := range paths {
		if len(paths) > 1 {
			fmt.Printf("%s:\n", path)
		}
		if err := listDir(path); err != nil {
			log.Printf("gls: cannot access '%s': %v\n", path, err)
		}
	}
}

func listDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	minBytes, err := parseSize(minSize)
	if err != nil {
		return fmt.Errorf("invalid min-size: %v", err)
	}
	maxBytes, err := parseSize(maxSize)
	if err != nil {
		return fmt.Errorf("invalid max-size: %v", err)
	}

	var filtered []fs.DirEntry
	for _, entry := range entries {
		if !showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if minBytes != -1 || maxBytes != -1 {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			size := info.Size()
			if minBytes != -1 && size < minBytes {
				continue
			}
			if maxBytes != -1 && size > maxBytes {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	sortEntries(filtered)

	if longFormat {
		printLong(path, filtered)
	} else {
		printShort(filtered)
	}
	return nil
}

func sortEntries(entries []fs.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		e1, e2 := entries[i], entries[j]
		
		if dirsFirst {
			if e1.IsDir() && !e2.IsDir() {
				return true
			}
			if !e1.IsDir() && e2.IsDir() {
				return false
			}
		}

		info1, _ := e1.Info()
		info2, _ := e2.Info()

		switch sortField {
		case "size":
			return info1.Size() > info2.Size()
		case "time":
			return info1.ModTime().After(info2.ModTime())
		case "ext":
			return getExt(e1.Name()) < getExt(e2.Name())
		default: // name
			return strings.ToLower(e1.Name()) < strings.ToLower(e2.Name())
		}
	})
}

func getExt(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func printShort(entries []fs.DirEntry) {
	for _, entry := range entries {
		printEntry(entry, "")
		fmt.Print("  ")
	}
	fmt.Println()
}

func printLong(path string, entries []fs.DirEntry) {
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		stat := info.Sys().(*syscall.Stat_t)
		uid := strconv.Itoa(int(stat.Uid))
		gid := strconv.Itoa(int(stat.Gid))
		
		u, err := user.LookupId(uid)
		if err == nil {
			uid = u.Username
		}
		g, err := user.LookupGroupId(gid)
		if err == nil {
			gid = g.Name
		}

		modTime := info.ModTime().Format("Jan 02 15:04")
		perms := info.Mode().String()
		
		fmt.Printf("%s %s %s %8d %s ", perms, uid, gid, info.Size(), modTime)
		printEntry(entry, "")
		fmt.Println()
	}
}

func printEntry(entry fs.DirEntry, suffix string) {
	name := entry.Name()
	icon := ""
	color := White

	if showIcons {
		icon = getIcon(entry) + " "
	}

	if entry.IsDir() {
		color = Blue + Bold
	} else {
		info, _ := entry.Info()
		if info.Mode()&0111 != 0 {
			color = Green + Bold
		} else {
			ext := getExt(name)
			if c, ok := extensionColors[ext]; ok {
				color = c
			}
		}
	}

	fmt.Printf("%s%s%s%s%s", color, icon, name, Reset, suffix)
}

func getIcon(entry fs.DirEntry) string {
	if entry.IsDir() {
		return ""
	}
	name := entry.Name()
	if icon, ok := specialFileIcons[name]; ok {
		return icon
	}
	
	ext := getExt(name)
	if icon, ok := extensionIcons[ext]; ok {
		return icon
	}
	
	return ""
}

func parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return -1, nil
	}
	sizeStr = strings.ToUpper(sizeStr)
	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "K") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "K")
	} else if strings.HasSuffix(sizeStr, "M") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "M")
	} else if strings.HasSuffix(sizeStr, "G") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "G")
	}
	
	val, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return val * multiplier, nil
}

func loadConfig() Config {
	config := Config{}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return config
	}

	configPath := filepath.Join(configDir, "gls", "config.json")
	file, err := os.ReadFile(configPath)
	if err != nil {
		return config
	}

	if err := json.Unmarshal(file, &config); err != nil {
		log.Printf("gls: error parsing config file: %v\n", err)
	}
	return config
}

