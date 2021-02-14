package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/go-playground/validator.v10"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"time"
)

type tmplData struct {
	Phone           string `json:"phone"`
	Address         string `json:"address"`
	Email           string `json:"email"`
	Whatsapp        string `json:"whatsapp"`
	Instagram       string `json:"instagram"`
	Facebook        string `json:"facebook"`
	Tiktok          string `json:"tiktok"`
	YandexMapAPIKey string `json:"yandex_map_api_key"`
}

type Conf struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`

	Email struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Email    string `json:"email"`
	} `json:"email"`

	TmplData tmplData `json:"tmpl_data"`
}

type Service struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Price string `json:"price"`
}

type Order struct {
	Service string `validate:"required"`
	Name    string `validate:"required"`
	Phone   string
	Email   string `validate:"required,email"`
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func sendEmail(conf *Conf, order *Order) {
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Password, conf.Email.Host)

	to := []string{conf.Email.Email}
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=\"utf-8\"\r\n"+
		"\r\n"+
		"%s\n%s\n%s\r\n", conf.Email.Email, order.Service, order.Name, order.Phone, order.Email))
	err := smtp.SendMail(fmt.Sprintf("%s:%s", conf.Email.Host, conf.Email.Port), auth, conf.Email.Email, to, msg)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	// Hello Kisa Kisovna, the web server
	AppIp := os.Getenv("APP_IP")
	AppPort := os.Getenv("APP_PORT")

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	f, err := ioutil.ReadFile("conf.json")
	check(err)

	conf := Conf{Host: AppIp, Port: AppPort}
	err = json.Unmarshal(f, &conf)
	check(err)

	masterTmpl, _ := template.ParseFiles("templates/base.gohtml")

	validate := validator.New()

	indexHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		if r.Method == http.MethodPost {
			_ = r.ParseForm()

			order := Order{r.Form["service"][0], r.Form["name"][0], r.Form["phone"][0], r.Form["email"][0]}
			if err = validate.Struct(order); err != nil {
				go sendEmail(&conf, &order)
			}

			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusSeeOther) // Django return http.StatusFound
			return
		}

		tmpl, err := template.Must(masterTmpl.Clone()).ParseFiles("templates/index.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s, _ := ioutil.ReadFile("services.json")
		var services []Service
		_ = json.Unmarshal(s, &services)

		var gallery []string
		err = filepath.Walk("./static/img/gallery", func(path string, info os.FileInfo, err error) error {
			if info.IsDir() || filepath.Ext(path) != ".jpg" {
				return nil
			}
			gallery = append(gallery, info.Name())
			return nil
		})

		data := struct {
			Contacts tmplData
			Services []Service
			Gallery  []string
		}{conf.TmplData, services, gallery}

		err = tmpl.ExecuteTemplate(w, "layout", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", conf.Host, conf.Port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
