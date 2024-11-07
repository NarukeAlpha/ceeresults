package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

var IphoneUserAgentList = []string{
	"iPhone 6", "iPhone 6 plus",
	"iPhone 7", "iPhone 7 plus",
	"iPhone 8", "iPhone 8 plus",
	"iPhone X", "iPhone XR",
	"iPhone XS", "iPhone XS Max",
	"iPhone 11", "iPhone 11 Pro", "iPhone 11 Pro Max",
	"iPhone SE (2nd generation)",
	"iPhone 12 mini", "iPhone 12", "iPhone 12 Pro", "iPhone 12 Pro Max",
	"iPhone 13 mini", "iPhone 13", "iPhone 13 Pro", "iPhone 13 Pro Max",
	"iPhone 14 mini", "iPhone 14", "iPhone 14 Pro", "iPhone 14 Pro Max",
}

type ProxyStruct struct {
	ip  string
	usr string
	pw  string
}

func AssertErrorToNil(message string, err error) {
	if err != nil {
		log.Panicf(message, err)
	}
}

type Resultados struct {
	Gobernador   Gobernador   `json:"gobernador"`
	Informacion  Informacion  `json:"informacion"`
	Webby        Webby        `json:"webby"`
	ComResidente ComResidente `json:"comresidente"`
}
type Webby struct {
	UrlG string `json:"urlg"`
	UrlC string `json:"urlc"`
}

type Gobernador struct {
	Jenniffervotes   int     `json:"jenniffervotes"`
	Jennifferpercent float64 `json:"jennifferpercent"`
	// 21 and 22 respectively
	Jesusvotes   int     `json:"jesusvotes"`
	Jesuspercent float64 `json:"jesuspercent"`
	//26 and 27 respectively
	JavierCvotes   int     `json:"javiervotes"`
	JavierCpercent float64 `json:"javierpercent"`
	//31 and 32 respectively
	Juanvotes   int     `json:"juanvotes"`
	Juanpercent float64 `json:"juanpercent"`
	//35 and 36 respectively
	JavierJvotes   int     `json:"javier2votes"`
	JavierJpercent float64 `json:"javier2percent"`
	//40 and 41 respectively
}
type Informacion struct {
	Participacion float64 `json:"participacion"`
	//76 in slice
	NominacionDirecta int `json:"nominaciondirecta"`
	//51 in slice
	TotalDePapeletas int `json:"totaldepapeletas"`
	//74 in slice
	ColegiosReportados int `json:"colegiosreportados"`
	//124 in slice off 5408
	ColegiosRegulares int `json:"colegiosregulares"`
	//132 in slice off 4490
	ColegiosDeVotoAdelantado int `json:"colegiosdevotoadelantado"`
	//144 in slice of 918
}

type ComResidente struct {
	Williamvotes   int     `json:"williamvotes"`
	Williampercent float64 `json:"williampercent"`
	//22 and 23
	Pablovotes   int     `json:"pablovotes"`
	Pablopercent float64 `json:"pablopercent"`
	//28 and 29
	anaVotes   int     `json:"anavotes"`
	AnaPercent float64 `json:"anapercent"`
	//34 and 35
	robertoVotes   int     `json:"robertovotes"`
	RobertoPercent float64 `json:"robertopercent"`
	//40 and 41
	vivianaVotes   int     `json:"vivianavotes"`
	VivianaPercent float64 `json:"vivianapercent"`
	//45 and 46
}

var resultados Resultados
var mw io.Writer

func init() {
	_, err := os.Stat("data.json")
	if os.IsNotExist(err) {
		_, err = os.Create("data.json")
		if err != nil {
			log.Fatal(err)
		}
	}
	_, err = os.Stat("ProxyList.csv")
	if os.IsNotExist(err) {
		log.Fatalln("PoxyList.csv not found, please provide a csv file named ProxyList in the same directory as the exe")

	}
	file, err := os.Open("data.json")
	if err != nil {
		log.Panicf("Error opening data.json: %v", err)

	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&resultados)
	log.Println(resultados)
	if err == io.EOF {
		log.Println("No data in data.json")
	} else if err != nil {
		log.Panicf("Error decoding data.json: %v", err)
	}

	os.Setenv("DISPLAY", ":10") // or whatever your display number is
	err = playwright.Install()
	if err != nil {
		log.Fatal("Could not install Playwright")
	}
	log.Println("Initializing")

	_, err = os.Stat("log.txt")
	if os.IsNotExist(err) {
		_, err = os.Create("log.txt")
		if err != nil {
			log.Fatal(err)
		}
	}
	file, err = os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	mw = io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)
	log.Println("Started successfully")
	log.Println(resultados)
}

func main() {

	log.Println("Loading Proxies")
	pChannel := make(chan []ProxyStruct)
	go proxyList(pChannel)
	PL := <-pChannel
	close(pChannel)
	log.Println("Proxies Loaded & Starting Tasks")

	TaskInit(PL)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop
}

func TaskInit(pL []ProxyStruct) {
	errch := make(chan error)
	var err error

	for {
		for _, proxy := range pL {
			go Task(proxy, errch)
			err = <-errch
			if err != nil {
				//if the task panics at any point it will be caught here and the task will be restarted
				continue
			} else {
				time.Sleep(1 * time.Minute)
			}
		}
	}

}

func PlaywrightInit(proxy ProxyStruct, pw *playwright.Playwright) (playwright.BrowserContext, error) {

	device := pw.Devices[IphoneUserAgentList[rand.Intn(len(IphoneUserAgentList)-1)]]
	pwProxyStrct := playwright.Proxy{
		Server:   proxy.ip,
		Username: &proxy.usr,
		Password: &proxy.pw,
	}

	browser, err := pw.Chromium.LaunchPersistentContext("", playwright.BrowserTypeLaunchPersistentContextOptions{
		Viewport:          device.Viewport,
		UserAgent:         playwright.String(device.UserAgent),
		DeviceScaleFactor: playwright.Float(device.DeviceScaleFactor),
		IsMobile:          playwright.Bool(device.IsMobile),
		HasTouch:          playwright.Bool(device.HasTouch),
		Headless:          playwright.Bool(false),
		//RecordHarContent: playwright.HarContentPolicyAttach,
		//RecordHarMode: playwright.HarModeFull,
		//RecordHarPath: playwright.String("test.har"),

		ColorScheme: playwright.ColorSchemeDark,
		Proxy:       &pwProxyStrct,
		IgnoreDefaultArgs: []string{
			"--enable-automation",
		},
	})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}

	script := playwright.Script{
		Content: playwright.String(`
    const defaultGetter = Object.getOwnPropertyDescriptor(
      Navigator.prototype,
      "webdriver"
    ).get;
    defaultGetter.apply(navigator);
    defaultGetter.toString();
    Object.defineProperty(Navigator.prototype, "webdriver", {
      set: undefined,
      enumerable: true,
      configurable: true,
      get: new Proxy(defaultGetter, {
        apply: (target, thisArg, args) => {
          Reflect.apply(target, thisArg, args);
          return false;
        },
      }),
    });
    const patchedGetter = Object.getOwnPropertyDescriptor(
      Navigator.prototype,
      "webdriver"
    ).get;
    patchedGetter.apply(navigator);
    patchedGetter.toString();
  `),
	}
	err = browser.AddInitScript(script)
	if err != nil {
		log.Fatalf("could not add initialization script: %v", err)
	}

	log.Printf("Browser Launched, user agent: %v, Proxy: %v : %v \n", device, proxy.ip, proxy.pw)
	log.Println()
	return browser, nil
}

func Task(proxy ProxyStruct, errch chan error) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered from panic: %v assuming bad proxy", err)
			time.Sleep(5 * time.Second)
			err := errors.New("recovered from panic")
			errch <- err
		}
	}()
	log.Println("Initializing playwright instance")
	PlaywrightInstance, err := playwright.Run()
	defer func(PlaywrightInstance *playwright.Playwright) {
		err = PlaywrightInstance.Stop()
		if err != nil {
			log.Panicf("could not stop playwright: %v", err)
		}
	}(PlaywrightInstance)
	browser, err := PlaywrightInit(proxy, PlaywrightInstance)
	defer func(browser playwright.BrowserContext, options ...playwright.BrowserContextCloseOptions) {
		err = browser.Close()
		if err != nil {
			log.Panicf("could not close browser: %v", err)
		}
	}(browser)
	if err != nil {
		log.Panicf("could not initialize playwright: %v", err)
	}
	log.Println("opening a new page")
	page, err := browser.NewPage()
	if err != nil {
		log.Panicf("could not create page: %v", err)
	}
	defer func(page playwright.Page, options ...playwright.PageCloseOptions) {
		err = page.Close()
		if err != nil {
			log.Panicf("could not close page: %v", err)
		}
	}(page)
	page2, err := browser.NewPage()
	if err != nil {
		log.Panicf("could not create second page: %v", err)
	}
	defer func(page2 playwright.Page, options ...playwright.PageCloseOptions) {
		err = page2.Close()
		if err != nil {
			log.Panicf("could not close page: %v", err)

		}
	}(page2)
	log.Println("navigating to page")
	if _, err = page.Goto("https://elecciones2024.ceepur.org/Noche_del_Evento_120/index.html#es/default/GOBERNADOR_Resumen.xml"); err != nil {
		log.Panicf("could not navigate to page: %v", err)

	}
	if _, err = page2.Goto("https://elecciones2024.ceepur.org/Noche_del_Evento_120/index.html#es/default/COMISIONADO_RESIDENTE_Resumen.xml"); err != nil {
		log.Panicf("could not navigate to page: %v", err)
	}
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		log.Printf("network idle state not reached: %v", err)
	}

	// Get the innerHTML of the element with class "content"
	content, err := page.InnerText("section.content")
	if err != nil {
		log.Panicf("could not get innerHTML: %v", err)
	}

	// Remove all \n and \t characters
	re := regexp.MustCompile(`[\n\t]`)
	cleanedText := re.ReplaceAllString(content, " ")

	// Split the cleaned string into slices
	slices := strings.Fields(cleanedText)

	val, err := parseInt(slices[74])
	if resultados.Informacion.TotalDePapeletas < val {
		err = page2.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})
		if err != nil {
			log.Printf("network idle state not reached: %v", err)
		}
		content2, err := page2.InnerText("section.content")
		if err != nil {
			log.Panicf("could not get innerHTML: %v", err)
		}
		cleanedText2 := re.ReplaceAllString(content2, " ")
		slices2 := strings.Fields(cleanedText2)

		resultados.Informacion.TotalDePapeletas = val
		resultados.Informacion.Participacion, err = parsePercentage(slices[76])
		log.Println(resultados.Informacion.Participacion)
		resultados.Informacion.NominacionDirecta, err = parseInt(slices[51])
		resultados.Informacion.ColegiosReportados, err = parseInt(slices[124])
		resultados.Informacion.ColegiosRegulares, err = parseInt(slices[132])
		resultados.Informacion.ColegiosDeVotoAdelantado, err = parseInt(slices[144])
		/*

		 */
		for i := 0; i < len(slices); i++ {
			if slices[i] == "González" {
				// Ensure there are enough slices after "Gonzales"
				if i+2 < len(slices) {
					// Convert the next slice to an integer for votes
					resultados.Gobernador.Jenniffervotes, err = parseInt(slices[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.Gobernador.Jennifferpercent, err = parsePercentage(slices[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices[i] == "Ortiz" {
				if i+2 < len(slices) {
					// Convert the next slice to an integer for votes
					resultados.Gobernador.Jesusvotes, err = parseInt(slices[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.Gobernador.Jesuspercent, err = parsePercentage(slices[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices[i] == "Dalmau" {
				if i+2 < len(slices) {
					// Convert the next slice to an integer for votes
					resultados.Gobernador.Juanvotes, err = parseInt(slices[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.Gobernador.Juanpercent, err = parsePercentage(slices[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices[i] == "Pérez" {
				if i+2 < len(slices) {
					// Convert the next slice to an integer for votes
					resultados.Gobernador.JavierJvotes, err = parseInt(slices[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.Gobernador.JavierJpercent, err = parsePercentage(slices[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices[i] == "Iturregui" {
				if i+2 < len(slices) {
					// Convert the next slice to an integer for votes
					resultados.Gobernador.JavierCvotes, err = parseInt(slices[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.Gobernador.JavierCpercent, err = parsePercentage(slices[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			}
		} //comisionado
		for i := 0; i < len(slices2); i++ {
			if slices2[i] == "Villafañe" {
				// Ensure there are enough slices after "Gonzales"
				if i+2 < len(slices2) {
					// Convert the next slice to an integer for votes
					resultados.ComResidente.Williamvotes, err = parseInt(slices2[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.ComResidente.Williampercent, err = parsePercentage(slices2[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices2[i] == "Hernández" {
				if i+2 < len(slices2) {
					// Convert the next slice to an integer for votes
					resultados.ComResidente.Pablovotes, err = parseInt(slices2[i+2])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.ComResidente.Pablopercent, err = parsePercentage(slices2[i+3])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices2[i] == "Lassén" {
				if i+2 < len(slices2) {
					// Convert the next slice to an integer for votes
					resultados.ComResidente.anaVotes, err = parseInt(slices2[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.ComResidente.AnaPercent, err = parsePercentage(slices2[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices2[i] == "Morales" {
				if i+2 < len(slices2) {
					// Convert the next slice to an integer for votes
					resultados.ComResidente.vivianaVotes, err = parseInt(slices2[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.ComResidente.VivianaPercent, err = parsePercentage(slices2[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			} else if slices2[i] == "Correa" {
				if i+2 < len(slices) {
					// Convert the next slice to an integer for votes
					resultados.ComResidente.robertoVotes, err = parseInt(slices2[i+1])
					if err != nil {
						log.Panicf("Error converting votes: %v", err)
					}

					// Convert the following slice to an integer for percent
					resultados.ComResidente.RobertoPercent, err = parsePercentage(slices2[i+2])
					if err != nil {
						log.Panicf("Error converting percent: %v", err)
					}
				} else {
					log.Panicf("Not enough data after 'Gonzales'")
				}
			}
		}
		log.Println(resultados)
		file, err := os.OpenFile("data.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Panicf("Error opening data.json: %v", err)
		}

		encoder := json.NewEncoder(file)
		if err = encoder.Encode(resultados); err != nil {
			log.Panicf("Error encoding data.json: %v", err)
		}
		file.Close()
		webhookGob(resultados)
		webhookCom(resultados)
	}
	log.Printf("finished task for proxy :%v", proxy.ip)
	errch <- nil

}

func parsePercentage(value string) (float64, error) {
	value = strings.TrimSuffix(value, "%")
	return strconv.ParseFloat(value, 64)
}
func parseInt(value string) (int, error) {
	value = strings.ReplaceAll(value, ",", "")
	return strconv.Atoi(value)
}
func proxyList(c chan []ProxyStruct) {
	var returnPS []ProxyStruct
	var path = "./ProxyList.csv"
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("couldn't open - err: %v", err)
	}

	csvReader := csv.NewReader(f)
	for i := 0; true; i++ {
		if i == 0 {
			fmt.Println("Loading proxies")
			_, err := csvReader.Read()
			if err != nil {
				log.Fatalf("failed to open csv - err: %v", err)
			}

		} else {
			rec, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatalf("CSV reader failed - err : %v", err)
			}
			fmt.Printf("%+v \n", rec)
			split := strings.Split(rec[0], ":")
			fmt.Printf(" proxy string %v \n", split)
			srv := (split[0] + ":" + split[1])
			usr := split[2]
			pss := split[3]

			var accDataStrct = ProxyStruct{
				ip:  srv,
				usr: usr,
				pw:  pss,
			}
			returnPS = append(returnPS, accDataStrct)

		}

	}
	err = f.Close()
	if err != nil {
		log.Fatalf("failed to close file - err: %v", err)
	}
	c <- returnPS
	return
}

func webhookGob(strct Resultados) {
	payload := map[string]interface{}{
		"content": "Results so far",
		"embeds": []map[string]interface{}{
			{
				"title": "Informacion General",
				"description": fmt.Sprintf("Participacion: %v\nNominacion Directa: %d\nTotal De Papeletas: %d\nColegios Reportados: %d de 5408\nColegios Regulares: %d de 4490\nColegios de Voto Adelantado: %d de 918",
					strct.Informacion.Participacion,
					strct.Informacion.NominacionDirecta,
					strct.Informacion.TotalDePapeletas,
					strct.Informacion.ColegiosReportados,
					strct.Informacion.ColegiosRegulares,
					strct.Informacion.ColegiosDeVotoAdelantado),
				"color": 0,
			},
			{
				"title":       "Jennifer Gonzales",
				"description": fmt.Sprintf("Votos: %d\nPorciento: %v%%", strct.Gobernador.Jenniffervotes, strct.Gobernador.Jennifferpercent),
				"color":       1652433,
			},
			{
				"title":       "Jesus Manuel Ortiz",
				"description": fmt.Sprintf("Votos: %d\nPorciento: %v%%", strct.Gobernador.Jesusvotes, strct.Gobernador.Jesuspercent),
				"color":       13568008,
			},
			{
				"title":       "Javier Cordova",
				"description": fmt.Sprintf("Votos: %d\nPorciento: %v%%", strct.Gobernador.JavierCvotes, strct.Gobernador.JavierCpercent),
				"color":       nil,
			},
			{
				"title":       "Juan Dalmau",
				"description": fmt.Sprintf("Votos: %d\nPorciento: %v%%", strct.Gobernador.Juanvotes, strct.Gobernador.Juanpercent),
				"color":       301316,
			},
			{
				"title":       "Javier Jimenez",
				"description": fmt.Sprintf("Votos: %d\nPorciento: %v%%", strct.Gobernador.JavierJvotes, strct.Gobernador.JavierJpercent),
				"color":       1349560,
			},
		},
		"attachments": []interface{}{},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Panicf("Error marshalling payload: %v", err)
	}
	url := resultados.Webby.UrlG

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Panicf("Error sending webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Webhook returned non-OK status: %v", resp.Status)
	}
}

func webhookCom(strct Resultados) {
	payload := map[string]interface{}{
		"content": "Results so far",
		"embeds": []map[string]interface{}{
			{
				"title": "Comisionado Residente",
				"description": fmt.Sprintf("William Villafane\nVotos: %v\nPorciento: %v%%\n\n\nPablo Vazquez\nVotos: %v\nPorciento: %v%%\n\n\nAna Irma Rivera Lassén\nVotos: %√\nPorciento: %√%%\n\n\nRoberto Prats\nVotos: %v\nPorciento: %v%%\n\n\nViviana Perez\nVotos: %√\nPorciento: %v%%",
					strct.ComResidente.Williamvotes, strct.ComResidente.Williampercent,
					strct.ComResidente.Pablovotes, strct.ComResidente.Pablopercent,
					strct.ComResidente.anaVotes, strct.ComResidente.AnaPercent,
					strct.ComResidente.robertoVotes, strct.ComResidente.RobertoPercent,
					strct.ComResidente.vivianaVotes, strct.ComResidente.VivianaPercent),
				"color": 0,
			},
		},
		"attachments": []interface{}{},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Panicf("Error marshalling payload: %v", err)
	}
	url := resultados.Webby.UrlC
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Panicf("Error sending webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Webhook returned non-OK status: %v", resp.Status)
	}
}
