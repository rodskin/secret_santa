package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"slices"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/jordan-wright/email"
)

const SENDMAIL = true

// Participant représente une personne avec un nom et un email
type Participant struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	CannotDraw []string `json:"cannotDraw"`
}

// Fonction pour mélanger la liste des participants
func shuffle(slice []Participant) {
	rand.Shuffle(len(slice), func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

// Vérifie si un participant peut tirer une autre personne en fonction des restrictions
func canDraw(participant Participant, drawn Participant) bool {
	if slices.Contains(drawn.CannotDraw, participant.Name) {
		return false
	}
	return true
}

// Vérifie si un tirage est valide
func isValidDraw(participants, shuffled []Participant) bool {
	for i := range participants {
		// Vérifie que le participant ne s'est pas tiré lui-même et respecte les restrictions
		if participants[i].Name == shuffled[i].Name || !canDraw(participants[i], shuffled[i]) {
			return false
		}
		// Vérifie qu'il n'y a pas de réciprocité
		for j := range participants {
			if participants[i].Name == shuffled[j].Name && participants[j].Name == shuffled[i].Name {
				return false
			}
		}
	}
	return true
}

// Fonction pour envoyer un email via SMTP
func sendEmail(from string, password string, to string, subject string, body string, html bool) error {
	e := email.NewEmail()
	e.From = from
	e.To = []string{to}
	e.Subject = subject
	if !html {
		e.Text = []byte(body)
	} else {
		e.HTML = []byte(body)
	}

	// Connexion SMTP (exemple pour Gmail)
	auth := smtp.PlainAuth("", from, password, "smtp.gmail.com")

	err := e.Send("smtp.gmail.com:587", auth)
	return err
}

// Lire les participants depuis un fichier JSON
func readParticipants(filename string) ([]Participant, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Lire le contenu du fichier avec le paquet os
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Décode le JSON en une liste de Participant
	var participants []Participant
	err = json.Unmarshal(bytes, &participants)
	if err != nil {
		return nil, err
	}

	return participants, nil
}

// Génère la correspondance du Secret Santa et envoie les emails
func secretSanta(participants []Participant, smtpUser, smtpPass string, sendMail bool, mailSubject string, mailBody string) {
	n := len(participants)

	// Créer une liste mélangée pour correspondre les cadeaux
	shuffled := make([]Participant, n)
	// Essayez de trouver un tirage valide
	for {
		copy(shuffled, participants)
		shuffle(shuffled)
		if isValidDraw(participants, shuffled) {
			break
		}
	}

	// Envoi des emails
	for i := range participants {
		if !sendMail {
			fmt.Printf("Email envoyé à %s pour leur Secret Santa : %s\n", participants[i].Name, shuffled[i].Name)
			continue
		}
		// Corps de l'email en HTML
		body := fmt.Sprintf(mailBody, participants[i].Name, shuffled[i].Name)

		err := sendEmail(smtpUser, smtpPass, participants[i].Email, mailSubject, body, true)
		if err != nil {
			log.Fatalf("Erreur lors de l'envoi de l'email à %s: %v", participants[i].Email, err)
		}

		fmt.Println("Email envoyé à", participants[i].Name)
	}
}

func main() {
	// Lire les participants depuis le fichier JSON
	participants, err := readParticipants("participants.json")
	if err != nil {
		log.Fatalf("Erreur lors de la lecture du fichier JSON: %v", err)
	}

	// Charger le fichier .env
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Erreur lors du chargement du fichier .env")
	}

	// Récupérer les variables d'environnement
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	sendMailStr := os.Getenv("SENDMAIL")
	sendMail, err := strconv.ParseBool(sendMailStr)
	if err != nil {
		log.Fatalf("Impossible de convertir SENDMAIL en bool: %v", err)
	}
	mailSubject := os.Getenv("MAIL_SUBJECT")

	content, err := os.ReadFile("mail_body.html")
	if err != nil {
		log.Fatal(err)
	}

	mailBody := string(content) // convertir en string

	// Générer le Secret Santa et envoyer les emails
	secretSanta(participants, smtpUser, smtpPass, sendMail, mailSubject, mailBody)
}
