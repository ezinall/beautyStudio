// Kisa Kisovna the Web-site

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/net/xsrftoken"
	"gopkg.in/go-playground/validator.v10"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type TmplData struct {
	Label    string `json:"label"`
	Contacts struct {
		Phone     string `json:"phone"`
		Address   string `json:"address"`
		Email     string `json:"email"`
		Whatsapp  string `json:"whatsapp"`
		Instagram string `json:"instagram"`
		Facebook  string `json:"facebook"`
		Tiktok    string `json:"tiktok"`
	} `json:"contacts"`
	Description     string `json:"description"`
	Keywords        string `json:"keywords"`
	YandexMetrika   int    `json:"yandexMetrika"`
	YandexMapAPIKey string `json:"yandexMapAPIKey"`
}

type Conf struct {
	SecretKey string `json:"secretKey"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	CertFile  string `json:"certFile"`
	KeyFile   string `json:"keyFile"`

	Email struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Email    string `json:"email"`
	} `json:"email"`
	To string `json:"to"`

	TmplData TmplData `json:"tmplData"`
}

var conf Conf

type Service struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Price string `json:"price"`
}

type Appointment struct {
	Service string `validate:"required"`
	Name    string `validate:"required"`
	Phone   string `validate:"required"`
	Email   string `validate:"email"`
}

var validate *validator.Validate

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func sendEmail(conf *Conf, order *Appointment) {
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Password, conf.Email.Host)

	to := []string{conf.To}
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=\"utf-8\"\r\n"+
		"\r\n"+
		"%s\n%s\n%s\r\n", conf.To, order.Service, order.Name, order.Phone, order.Email))
	err := smtp.SendMail(fmt.Sprintf("%s:%d", conf.Email.Host, conf.Email.Port), auth, conf.Email.Email, to, msg)
	if err != nil {
		log.Println(err)
	}
}

func appointmentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if xsrftoken.ValidFor(r.FormValue("token"), conf.SecretKey, "", "", xsrftoken.Timeout) {
		appointment := Appointment{
			r.FormValue("service"),
			r.FormValue("name"),
			r.FormValue("phone"),
			r.FormValue("email"),
		}
		if err := validate.Struct(appointment); err == nil {
			go sendEmail(&conf, &appointment)
		}
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound) // Django return http.StatusFound
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.gohtml")
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

	token := xsrftoken.Generate(conf.SecretKey, "", "")

	data := struct {
		TmplData
		Services []Service
		Gallery  []string
		Token    string
	}{conf.TmplData, services, gallery, token}

	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	f, err := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Println(err)
	} else {
		log.SetOutput(f)
	}
	defer f.Close()

	// Parse config ===================================================
	c, err := ioutil.ReadFile("conf.json")
	check(err)

	err = json.Unmarshal(c, &conf)
	check(err)

	// Set validator ==================================================
	validate = validator.New()

	// Create server ==================================================
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/appointment/", appointmentHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	s := &http.Server{
		//Addr:           fmt.Sprintf("%s:%s", os.Getenv("APP_IP"), os.Getenv("APP_PORT")),
		Addr:           fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	go func() {
		log.Fatal(s.ListenAndServe())
	}()

	// Graceful shutdown ==============================================
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = s.Shutdown(ctx); err != nil {
		log.Println(err)
	}
}
