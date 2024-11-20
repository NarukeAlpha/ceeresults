package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func main() {
	err := playwright.Install()
	if err != nil {
		log.Fatal("Could not install Playwright")
	}
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.LaunchPersistentContext("", playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(false),
	})
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	_, err = page.Goto("https://elecciones2024.ceepur.org/Escrutinio_General_121/index.html#es/pic_bar_list/SENADORES_POR_ACUMULACION_Resumen.xml")
	if err != nil {
		log.Fatalf("could not navigate to page: %v", err)
	}

	// Wait for the section.content element to be present and visible
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		log.Fatalf("network idle state not reached: %v", err)
	}

	// Get the innerHTML of the element with class "content"
	content, err := page.InnerText("section.content")
	if err != nil {
		log.Fatalf("could not get innerHTML: %v", err)
	}
	// Remove all \n and \t characters
	re := regexp.MustCompile(`[\n\t]`)
	cleanedText := re.ReplaceAllString(content, " ")

	// Split the cleaned string into slices
	slices := strings.Fields(cleanedText)

	// Print the slices
	for _, slice := range slices {
		fmt.Println(slice)
	}

	log.Println("Page content:", content)
}
