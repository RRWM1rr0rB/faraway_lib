package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// printTree рекурсивно выводит дерево файлов и папок
func printTree(root string, indent string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Println("Ошибка чтения директории:", err)
		return
	}

	for i, entry := range entries {
		connector := "├──"
		if i == len(entries)-1 {
			connector = "└──"
		}

		fmt.Println(indent + connector + " " + entry.Name())
		if entry.IsDir() {
			newIndent := indent + "│   "
			if i == len(entries)-1 {
				newIndent = indent + "    "
			}
			printTree(filepath.Join(root, entry.Name()), newIndent)
		}
	}
}

func main() {
	root := "." // Начальная директория (текущая папка)
	fmt.Println(root)
	printTree(root, "")
}
