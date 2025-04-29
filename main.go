package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	db           *sql.DB
	store        *sessions.CookieStore
	oauth2Config *oauth2.Config
)

// Custom types
type contextKey string

const emailKey contextKey = "email"

type NotFoundError struct {
	Resource string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Resource)
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("Validation error for %s: %s", e.Field, e.Message)
}

type DatabaseError struct {
	Operation string
	Err       error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("Database error during %s: %v", e.Operation, e.Err)
}

// Models
type User struct {
	ID              int
	Email           string
	Username        string
	DisplayName     string
	LifeAspirations sql.NullString
	ThingsILikeToDo sql.NullString
	ProfileImageURL sql.NullString
	Bio             sql.NullString
	BioLink         sql.NullString
	IsBanned        bool
}

type AspirationUpdate struct {
	ID              int
	Username        string
	DisplayName     string
	Content         string
	CreatedAt       time.Time
	LikeCount       int
	CommentCount    int
	Liked           bool
	IsOwnPost       bool
	ProfileImageURL sql.NullString
}

type Comment struct {
	ID              int
	UpdateID        int
	UserID          int
	ParentID        sql.NullInt64
	Content         string
	CreatedAt       time.Time
	Username        string
	DisplayName     string
	ProfileImageURL string
	Replies         []*Comment
}

type CommentContext struct {
	Root            interface{}
	Comment         *Comment
	UpdateID        int
	IsAuthenticated bool
}

type Administrator struct {
	ID       int
	Email    string
	Username string
}

// Main and setup
func main() {
	loadConfig()
	connectDB()
	defer db.Close()
	initializeOAuth()
	initializeSessionStore()
	setupRoutes()
	startServer()
}

func loadConfig() {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if googleClientID == "" || googleClientSecret == "" || googleRedirectURL == "" {
		log.Fatal("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL environment variables must be set")
	}

	// Initialize oauth2Config here
	oauth2Config = &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  googleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func connectDB() {
	postgresUrl := os.Getenv("POSTGRES_URL")
	if postgresUrl == "" {
		log.Fatal("POSTGRES_URL environment variable must be set")
	}

	var err error
	db, err = sql.Open("postgres", postgresUrl)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully connected to the database")
}

func initializeOAuth() {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if googleClientID == "" || googleClientSecret == "" || googleRedirectURL == "" {
		log.Fatal("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL environment variables must be set")
	}

	oauth2Config = &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  googleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func initializeSessionStore() {
	store = sessions.NewCookieStore([]byte("your-secret-key"))
}

func setupRoutes() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join("static", r.URL.Path)
		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		fs.ServeHTTP(w, r)
	})))
	http.HandleFunc("/", homepageHandler)
	http.HandleFunc("/browse", browseHandler)
	http.HandleFunc("/auth/google/login", googleLoginHandler)
	http.HandleFunc("/auth/google/callback", googleCallbackHandler)
	http.HandleFunc("/auth/logout", logoutHandler)
	http.HandleFunc("/profile", authMiddleware(profileHandler))
	http.HandleFunc("/profile/edit", authMiddleware(profileEditHandler))
	http.HandleFunc("/users/", publicProfileHandler)
	http.HandleFunc("/aspiration-update", authMiddleware(aspirationUpdateHandler))
	http.HandleFunc("/aspiration-update/edit/", authMiddleware(editAspirationUpdateHandler))
	http.HandleFunc("/aspiration-update/delete/", authMiddleware(deleteAspirationUpdateHandler))
	http.HandleFunc("/like", authMiddleware(likeHandler))
	http.HandleFunc("/unlike", authMiddleware(unlikeHandler))
	http.HandleFunc("/follow", authMiddleware(followHandler))
	http.HandleFunc("/unfollow", authMiddleware(unfollowHandler))
	http.HandleFunc("/update/", updatePermalinkHandler)
	http.HandleFunc("/comment/add", authMiddleware(addCommentHandler))
	// admin handlers
	http.HandleFunc("/admin/ban-user", adminAuthMiddleware(banUserHandler))
	http.HandleFunc("/admin/unban-user", adminAuthMiddleware(unbanUserHandler))
	// pages
	http.HandleFunc("/terms", pageHandler("pages/doc_terms.html"))
	http.HandleFunc("/privacy", pageHandler("pages/doc_privacy.html"))
	http.HandleFunc("/community-guidelines", pageHandler("pages/doc_community_guidelines.html"))
}

func startServer() {
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			log.Printf("Error getting session: %v", err)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		iemail, ok := session.Values["email"]
		if !ok || iemail == nil {
			log.Printf("No email in session")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		email, ok := iemail.(string)
		if !ok || email == "" {
			log.Printf("Invalid email in session")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		var isBanned bool
		err = db.QueryRow("SELECT is_banned FROM users WHERE email = $1", email).Scan(&isBanned)
		if err != nil {
			log.Printf("Error checking user ban status: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if isBanned {
			http.Error(w, "Your account has been banned", http.StatusForbidden)
			return
		}

		// Add the email to the request context
		ctx := context.WithValue(r.Context(), emailKey, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func adminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		email, ok := session.Values["email"].(string)
		if !ok || email == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM administrators WHERE email = $1", email).Scan(&count)
		if err != nil || count == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Helper functions
func getCurrentUserID(r *http.Request) int {
	email := r.Context().Value(emailKey).(string)
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		return 0
	}
	return userID
}

func fetchRecentUsers() ([]struct {
	ID              int
	Username        string
	DisplayName     string
	ProfileImageURL string
}, error,
) {
	rows, err := db.Query(`
		SELECT id, COALESCE(username, ''), COALESCE(display_name, ''), COALESCE(profile_image_url, '')
		FROM users
		WHERE is_banned = false
		ORDER BY id DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recentUsers []struct {
		ID              int
		Username        string
		DisplayName     string
		ProfileImageURL string
	}
	for rows.Next() {
		var user struct {
			ID              int
			Username        string
			DisplayName     string
			ProfileImageURL string
		}
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.ProfileImageURL); err != nil {
			return nil, err
		}
		recentUsers = append(recentUsers, user)
	}
	// It's okay if recentUsers is empty, we'll just return an empty slice
	return recentUsers, nil
}

func fetchRecentUpdates(currentUserID int, limit int) ([]AspirationUpdate, error) {
	query := `
        WITH RankedUpdates AS (
            SELECT au.id, u.username, u.display_name, u.profile_image_url, au.content, au.created_at, 
                   COUNT(DISTINCT l.id) as like_count,
                   COUNT(DISTINCT c.id) as comment_count,
                   CASE WHEN $1 > 0 AND EXISTS (SELECT 1 FROM likes WHERE user_id = $1 AND update_id = au.id) THEN TRUE ELSE FALSE END as liked,
                   CASE WHEN $1 > 0 AND au.user_id = $1 THEN TRUE ELSE FALSE END as is_own_post,
                   ROW_NUMBER() OVER (PARTITION BY au.user_id ORDER BY au.created_at DESC) as rn
            FROM aspiration_updates au
            JOIN users u ON au.user_id = u.id
            LEFT JOIN likes l ON au.id = l.update_id
            LEFT JOIN comments c ON au.id = c.update_id
            WHERE u.is_banned = false
            GROUP BY au.id, u.username, u.display_name, u.profile_image_url, au.content, au.created_at, au.user_id
        )
        SELECT id, COALESCE(username, ''), COALESCE(display_name, ''), COALESCE(profile_image_url, ''), content, created_at, 
               like_count, comment_count, liked, is_own_post
        FROM RankedUpdates
        WHERE rn = 1
        ORDER BY created_at DESC
        LIMIT $2
    `
	rows, err := db.Query(query, currentUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recentUpdates []AspirationUpdate
	for rows.Next() {
		var update AspirationUpdate
		if err := rows.Scan(&update.ID, &update.Username, &update.DisplayName, &update.ProfileImageURL, &update.Content, &update.CreatedAt, &update.LikeCount, &update.CommentCount, &update.Liked, &update.IsOwnPost); err != nil {
			return nil, err
		}
		recentUpdates = append(recentUpdates, update)
	}
	return recentUpdates, nil
}

func renderProfileEditPage(w http.ResponseWriter, user User, errorMessage string) {
	tmpl, err := template.ParseFiles("templates/profile_edit.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		User         User
		ErrorMessage string
	}{
		User:         user,
		ErrorMessage: errorMessage,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Generic page handler
func pageHandler(templateName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("templates/" + templateName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// OAuth handlers
func googleLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := oauth2Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get the session
	session, _ := store.Get(r, "session-name")

	code := r.URL.Query().Get("code")
	token, err := oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := oauth2Config.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	userData, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(userData, &userInfo); err != nil {
		http.Error(w, "Failed to parse user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	email := userInfo["email"].(string)
	profileImageURL := userInfo["picture"].(string)

	// Check if the user exists in the database
	var userID int
	var username string
	err = db.QueryRow("SELECT id, username FROM users WHERE email = $1", email).Scan(&userID, &username)
	if err != nil {
		if err == sql.ErrNoRows {
			// User doesn't exist, create a new user with a random username
			randomUsername := generateRandomUsername()
			_, err = db.Exec("INSERT INTO users (email, profile_image_url, username) VALUES ($1, $2, $3)", email, profileImageURL, randomUsername)
			if err != nil {
				http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Set session values
			session.Values["email"] = email
			session.Values["newUser"] = true
			err = session.Save(r, w)
			if err != nil {
				http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Redirect to profile edit page
			http.Redirect(w, r, "/profile/edit", http.StatusFound)
			return
		} else {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// User exists, update the profile image URL
	_, err = db.Exec("UPDATE users SET profile_image_url = $1 WHERE email = $2", profileImageURL, email)
	if err != nil {
		http.Error(w, "Failed to update profile image: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set session values and redirect to profile page
	session.Values["email"] = email
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["email"] = nil
	session.Options.MaxAge = -1
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Handlers
func homepageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/homepage.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "session-name")
	userLoggedIn := session.Values["email"] != nil

	var currentUserID int
	if userLoggedIn {
		email := session.Values["email"].(string)
		err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&currentUserID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// If err == sql.ErrNoRows, currentUserID will remain 0 (meaning the user is not logged in)
	}

	recentUsers, err := fetchRecentUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	recentUpdates, err := fetchRecentUpdates(currentUserID, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		UserLoggedIn bool
		RecentUsers  []struct {
			ID              int
			Username        string
			DisplayName     string
			ProfileImageURL string
		}
		RecentUpdates []AspirationUpdate
	}{
		UserLoggedIn:  userLoggedIn,
		RecentUsers:   recentUsers,
		RecentUpdates: recentUpdates,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func browseHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/browse.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "session-name")
	userLoggedIn := session.Values["email"] != nil

	var currentUserID int
	if userLoggedIn {
		email := session.Values["email"].(string)
		err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&currentUserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	users, err := fetchRecentUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Modify the users slice to handle ProfileImageURL
	for i, user := range users {
		if user.ProfileImageURL != "" {
			users[i].ProfileImageURL = user.ProfileImageURL
		} else {
			users[i].ProfileImageURL = "" // Set a default value or leave it empty
		}
	}

	recentUpdates, err := fetchRecentUpdates(currentUserID, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Modify the recentUpdates slice to handle ProfileImageURL
	for i, update := range recentUpdates {
		if update.ProfileImageURL.Valid {
			recentUpdates[i].ProfileImageURL.String = update.ProfileImageURL.String
		} else {
			recentUpdates[i].ProfileImageURL.String = "" // Set a default value or leave it empty
		}
	}

	data := struct {
		UserLoggedIn  bool
		RecentUpdates []AspirationUpdate
		Users         []struct {
			ID              int
			Username        string
			DisplayName     string
			ProfileImageURL string
		}
	}{
		UserLoggedIn:  userLoggedIn,
		RecentUpdates: recentUpdates,
		Users:         users,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getProfileData(userID int, currentUserID int, page int, pageSize int) (ProfileData, error) {
	offset := (page - 1) * pageSize

	var user User
	err := db.QueryRow(`
		SELECT id, email, COALESCE(username, ''), COALESCE(display_name, ''), 
		life_aspirations, things_i_like_to_do, profile_image_url, bio, bio_link, is_banned 
		FROM users WHERE id = $1`, userID).Scan(
		&user.ID, &user.Email, &user.Username, &user.DisplayName, &user.LifeAspirations,
		&user.ThingsILikeToDo, &user.ProfileImageURL, &user.Bio, &user.BioLink, &user.IsBanned)
	if err != nil {
		return ProfileData{}, err
	}

	rows, err := db.Query(`
		SELECT au.id, au.content, au.created_at, COUNT(DISTINCT l.id) as like_count,
			   COUNT(DISTINCT c.id) as comment_count,
			   CASE WHEN EXISTS (SELECT 1 FROM likes WHERE user_id = $1 AND update_id = au.id) THEN TRUE ELSE FALSE END as liked,
			   au.user_id = $1 as is_own_post,
			   u.profile_image_url
		FROM aspiration_updates au
		JOIN users u ON au.user_id = u.id
		LEFT JOIN likes l ON au.id = l.update_id
		LEFT JOIN comments c ON au.id = c.update_id
		WHERE au.user_id = $2
		GROUP BY au.id, au.content, au.created_at, au.user_id, u.profile_image_url
		ORDER BY au.created_at DESC
		LIMIT $3 OFFSET $4
	`, currentUserID, userID, pageSize, offset)
	if err != nil {
		return ProfileData{}, err
	}
	defer rows.Close()

	var updates []struct {
		AspirationUpdate
		IsOwnPost bool
	}
	for rows.Next() {
		var update AspirationUpdate
		var isOwnPost bool
		if err := rows.Scan(&update.ID, &update.Content, &update.CreatedAt, &update.LikeCount, &update.CommentCount, &update.Liked, &isOwnPost, &update.ProfileImageURL); err != nil {
			return ProfileData{}, err
		}
		updates = append(updates, struct {
			AspirationUpdate
			IsOwnPost bool
		}{update, isOwnPost})
	}

	var isFollowing bool
	if currentUserID != 0 {
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM followers 
				WHERE follower_id = $1 AND followed_id = $2
			)
		`, currentUserID, userID).Scan(&isFollowing)
		if err != nil {
			return ProfileData{}, err
		}
	}

	var followerCount int
	err = db.QueryRow("SELECT COUNT(*) FROM followers WHERE followed_id = $1", userID).Scan(&followerCount)
	if err != nil {
		return ProfileData{}, err
	}

	rows, err = db.Query(`
		SELECT u.username, COALESCE(u.profile_image_url, '') as profile_image_url
		FROM followers f
		JOIN users u ON f.follower_id = u.id
		WHERE f.followed_id = $1
		ORDER BY f.created_at DESC
		LIMIT 10
	`, userID)
	if err != nil {
		return ProfileData{}, err
	}
	defer rows.Close()

	var recentFollowers []struct {
		Username        string
		ProfileImageURL string
	}
	for rows.Next() {
		var follower struct {
			Username        string
			ProfileImageURL string
		}
		if err := rows.Scan(&follower.Username, &follower.ProfileImageURL); err != nil {
			return ProfileData{}, err
		}
		recentFollowers = append(recentFollowers, follower)
	}

	var totalUpdates int
	err = db.QueryRow("SELECT COUNT(*) FROM aspiration_updates WHERE user_id = $1", userID).Scan(&totalUpdates)
	if err != nil {
		return ProfileData{}, err
	}
	totalPages := (totalUpdates + pageSize - 1) / pageSize

	return ProfileData{
		User:            user,
		Updates:         updates,
		IsFollowing:     isFollowing,
		FollowerCount:   followerCount,
		RecentFollowers: recentFollowers,
		CurrentPage:     page,
		TotalPages:      totalPages,
		PreviousPage:    page - 1,
		NextPage:        page + 1,
	}, nil
}

type ProfileData struct {
	User    User
	Updates []struct {
		AspirationUpdate
		IsOwnPost bool
	}
	IsFollowing     bool
	FollowerCount   int
	RecentFollowers []struct {
		Username        string
		ProfileImageURL string
	}
	CurrentPage  int
	TotalPages   int
	PreviousPage int
	NextPage     int
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, (&DatabaseError{Operation: "fetching user ID", Err: err}).Error(), http.StatusInternalServerError)
		return
	}

	page := 1
	pageSize := 5
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if _, err := fmt.Sscanf(pageParam, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}

	profileData, err := getProfileData(userID, userID, page, pageSize)
	if err != nil {
		http.Error(w, (&DatabaseError{Operation: "fetching profile data", Err: err}).Error(), http.StatusInternalServerError)
		return
	}

	if profileData.User.Username == "" {
		http.Redirect(w, r, "/profile/edit", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/profile.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, profileData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func banUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE users SET is_banned = true WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Failed to ban user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/users/"+r.FormValue("username"), http.StatusSeeOther)
}

func unbanUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE users SET is_banned = false WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Failed to unban user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/users/"+r.FormValue("username"), http.StatusSeeOther)
}

func publicProfileHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Path, "/users/")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	var userID int
	var isBanned bool
	err := db.QueryRow("SELECT id, is_banned FROM users WHERE username = $1", username).Scan(&userID, &isBanned)
	if err == sql.ErrNoRows {
		log.Printf("User not found: %s", username)
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Printf("Database error when fetching user %s: %v", username, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "session-name")
	currentUserEmail, _ := session.Values["email"].(string)
	isLoggedIn := currentUserEmail != ""

	var isAdmin bool
	if isLoggedIn {
		err = db.QueryRow("SELECT COUNT(*) > 0 FROM administrators WHERE email = $1", currentUserEmail).Scan(&isAdmin)
		if err != nil {
			log.Printf("Error checking admin status: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	if isBanned && !isAdmin {
		http.Error(w, "This user's profile is not available", http.StatusForbidden)
		return
	}

	var currentUserID int
	if isLoggedIn {
		err = db.QueryRow("SELECT id FROM users WHERE email = $1", currentUserEmail).Scan(&currentUserID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Database error when fetching current user ID: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	page := 1
	pageSize := 5
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if _, err := fmt.Sscanf(pageParam, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}

	profileData, err := getProfileData(userID, currentUserID, page, pageSize)
	if err != nil {
		log.Printf("Error fetching profile data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/public_profile.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		ProfileData
		IsOwnProfile bool
		IsLoggedIn   bool
		IsAdmin      bool
		IsBanned     bool
	}{
		ProfileData:  profileData,
		IsOwnProfile: profileData.User.Email == currentUserEmail,
		IsLoggedIn:   isLoggedIn,
		IsAdmin:      isAdmin,
		IsBanned:     isBanned,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func profileEditHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	if r.Method == "POST" {
		username := strings.ToLower(r.FormValue("username"))
		displayName := r.FormValue("display_name")
		lifeAspirations := r.FormValue("life_aspirations")
		thingsILikeToDo := r.FormValue("things_i_like_to_do")
		bio := r.FormValue("bio")
		bioLink := r.FormValue("bio_link")

		if username == "" {
			renderProfileEditPage(w, User{Email: email}, "Username cannot be empty")
			return
		}

		var currentUsername sql.NullString
		err := db.QueryRow("SELECT username FROM users WHERE email = $1", email).Scan(&currentUsername)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if username != currentUsername.String {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1 AND email != $2", username, email).Scan(&count)
			if err != nil {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if count > 0 {
				renderProfileEditPage(w, User{Email: email, Username: username}, "Username is already taken")
				return
			}
		}

		// Update the user's username, display name, life aspirations, things I like to do, bio, and bio link
		_, err = db.Exec("UPDATE users SET username = $1, display_name = $2, life_aspirations = $3, things_i_like_to_do = $4, bio = $5, bio_link = $6 WHERE email = $7",
			username,
			displayName,
			sql.NullString{String: lifeAspirations, Valid: lifeAspirations != ""},
			sql.NullString{String: thingsILikeToDo, Valid: thingsILikeToDo != ""},
			sql.NullString{String: bio, Valid: bio != ""},
			sql.NullString{String: bioLink, Valid: bioLink != ""},
			email)
		if err != nil {
			http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	var user User
	err := db.QueryRow("SELECT email, COALESCE(username, ''), COALESCE(display_name, ''), life_aspirations, things_i_like_to_do, profile_image_url, bio, bio_link FROM users WHERE email = $1", email).Scan(&user.Email, &user.Username, &user.DisplayName, &user.LifeAspirations, &user.ThingsILikeToDo, &user.ProfileImageURL, &user.Bio, &user.BioLink)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderProfileEditPage(w, user, "")
}

func aspirationUpdateHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	if r.Method == "POST" {
		content := r.FormValue("content")
		updateID := r.FormValue("update_id")

		if content == "" {
			http.Error(w, (&ValidationError{Field: "content", Message: "Content cannot be empty"}).Error(), http.StatusBadRequest)
			return
		}

		var userID int
		err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
		if err != nil {
			http.Error(w, (&DatabaseError{Operation: "fetching user ID", Err: err}).Error(), http.StatusInternalServerError)
			return
		}

		if updateID != "" {
			// If updateID is provided, update the existing aspiration update
			_, err = db.Exec("UPDATE aspiration_updates SET content = $1 WHERE id = $2 AND user_id = $3", content, updateID, userID)
			if err != nil {
				http.Error(w, (&DatabaseError{Operation: "updating aspiration update", Err: err}).Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// If no updateID is provided, create a new aspiration update
			_, err = db.Exec("INSERT INTO aspiration_updates (user_id, content) VALUES ($1, $2)", userID, content)
			if err != nil {
				http.Error(w, (&DatabaseError{Operation: "creating aspiration update", Err: err}).Error(), http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func editAspirationUpdateHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	updateID := strings.TrimPrefix(r.URL.Path, "/aspiration-update/edit/")
	if updateID == "" {
		http.NotFound(w, r)
		return
	}

	var update AspirationUpdate
	err := db.QueryRow("SELECT id, content FROM aspiration_updates WHERE id = $1 AND user_id = (SELECT id FROM users WHERE email = $2)", updateID, email).Scan(&update.ID, &update.Content)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch aspiration update: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		content := r.FormValue("content")
		if content == "" {
			http.Error(w, "Content cannot be empty", http.StatusBadRequest)
			return
		}

		_, err = db.Exec("UPDATE aspiration_updates SET content = $1 WHERE id = $2", content, updateID)
		if err != nil {
			http.Error(w, "Failed to update aspiration update: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit_aspiration_update.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		UpdateID string
		Content  string
	}{
		UpdateID: updateID,
		Content:  update.Content,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deleteAspirationUpdateHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	updateID := strings.TrimPrefix(r.URL.Path, "/aspiration-update/delete/")
	if updateID == "" {
		http.NotFound(w, r)
		return
	}

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete associated comments first
	_, err = tx.Exec("DELETE FROM comments WHERE update_id = $1", updateID)
	if err != nil {
		http.Error(w, "Failed to delete associated comments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete associated likes
	_, err = tx.Exec("DELETE FROM likes WHERE update_id = $1", updateID)
	if err != nil {
		http.Error(w, "Failed to delete associated likes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Now delete the aspiration update
	_, err = tx.Exec("DELETE FROM aspiration_updates WHERE id = $1 AND user_id = (SELECT id FROM users WHERE email = $2)", updateID, email)
	if err != nil {
		http.Error(w, "Failed to delete aspiration update: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func likeHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	updateID := r.FormValue("update_id")
	if updateID == "" {
		http.Error(w, "Missing update_id", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "Failed to get user ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the user is trying to like their own post
	var postOwnerID int
	err = db.QueryRow("SELECT user_id FROM aspiration_updates WHERE id = $1", updateID).Scan(&postOwnerID)
	if err != nil {
		http.Error(w, "Failed to get post owner: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if userID == postOwnerID {
		http.Error(w, "You cannot like your own post", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("INSERT INTO likes (user_id, update_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, updateID)
	if err != nil {
		http.Error(w, "Failed to add like: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func unlikeHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	updateID := r.FormValue("update_id")
	if updateID == "" {
		http.Error(w, "Missing update_id", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "Failed to get user ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM likes WHERE user_id = $1 AND update_id = $2", userID, updateID)
	if err != nil {
		http.Error(w, "Failed to remove like: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func followHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Missing username", http.StatusBadRequest)
		return
	}

	var followerID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&followerID)
	if err != nil {
		http.Error(w, "Failed to get follower ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var followedID int
	err = db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&followedID)
	if err != nil {
		http.Error(w, "Failed to get followed ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if followerID == followedID {
		http.Error(w, "You cannot follow yourself", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("INSERT INTO followers (follower_id, followed_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", followerID, followedID)
	if err != nil {
		http.Error(w, "Failed to follow user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func unfollowHandler(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value(emailKey).(string)

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Missing username", http.StatusBadRequest)
		return
	}

	var followerID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&followerID)
	if err != nil {
		http.Error(w, "Failed to get follower ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var followedID int
	err = db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&followedID)
	if err != nil {
		http.Error(w, "Failed to get followed ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM followers WHERE follower_id = $1 AND followed_id = $2", followerID, followedID)
	if err != nil {
		http.Error(w, "Failed to unfollow user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func updatePermalinkHandler(w http.ResponseWriter, r *http.Request) {
	updateID := strings.TrimPrefix(r.URL.Path, "/update/")

	log.Printf("Handling update permalink for ID: %s", updateID)

	var update AspirationUpdate
	var currentUserID int
	var isBanned bool

	// Get the current user ID if the user is authenticated
	session, _ := store.Get(r, "session-name")
	if email, ok := session.Values["email"].(string); ok {
		err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&currentUserID)
		if err != nil {
			log.Printf("Error getting user ID: %v", err)
			// Don't return here, continue with currentUserID as 0
		}
	}

	query := `
        SELECT au.id, u.username, COALESCE(u.display_name, ''), COALESCE(u.profile_image_url, ''), au.content, au.created_at, 
               COUNT(l.id) as like_count,
               CASE WHEN $1 > 0 AND EXISTS (SELECT 1 FROM likes WHERE user_id = $1 AND update_id = au.id) THEN TRUE ELSE FALSE END as liked,
               CASE WHEN $1 > 0 AND au.user_id = $1 THEN TRUE ELSE FALSE END as is_own_post,
               u.is_banned
        FROM aspiration_updates au
        JOIN users u ON au.user_id = u.id
        LEFT JOIN likes l ON au.id = l.update_id
        WHERE au.id = $2
        GROUP BY au.id, u.username, u.display_name, u.profile_image_url, au.content, au.created_at, au.user_id, u.is_banned
    `

	err := db.QueryRow(query, currentUserID, updateID).Scan(
		&update.ID, &update.Username, &update.DisplayName, &update.ProfileImageURL, &update.Content,
		&update.CreatedAt, &update.LikeCount, &update.Liked, &update.IsOwnPost, &isBanned,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Update not found: %s", updateID)
			http.Error(w, "Update not found", http.StatusNotFound)
		} else {
			log.Printf("Error fetching update: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if isBanned {
		http.Error(w, "This update is not available", http.StatusForbidden)
		return
	}

	comments, err := getComments(updateID)
	if err != nil {
		http.Error(w, (&DatabaseError{Operation: "fetching comments", Err: err}).Error(), http.StatusInternalServerError)
		return
	}

	// Modify comments for banned users
	modifyBannedUserComments(comments)

	totalComments := countTotalComments(comments)

	data := struct {
		Update          AspirationUpdate
		Comments        []*Comment
		TotalComments   int
		IsAuthenticated bool
	}{
		Update:          update,
		Comments:        comments,
		TotalComments:   totalComments,
		IsAuthenticated: currentUserID != 0,
	}

	funcMap := template.FuncMap{
		"commentContext": func(root interface{}, comment *Comment) CommentContext {
			return CommentContext{
				Root:    root,
				Comment: comment,
				UpdateID: root.(struct {
					Update          AspirationUpdate
					Comments        []*Comment
					TotalComments   int
					IsAuthenticated bool
				}).Update.ID,
				IsAuthenticated: root.(struct {
					Update          AspirationUpdate
					Comments        []*Comment
					TotalComments   int
					IsAuthenticated bool
				}).IsAuthenticated,
			}
		},
	}

	tmpl, err := template.New("aspiration_update.html").Funcs(funcMap).ParseFiles("templates/aspiration_update.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		// At this point, we've already started writing the response, so we can't use http.Error
		// Instead, log the error and return
		return
	}
}

func modifyBannedUserComments(comments []*Comment) {
	for _, comment := range comments {
		var isBanned bool
		err := db.QueryRow("SELECT is_banned FROM users WHERE id = $1", comment.UserID).Scan(&isBanned)
		if err != nil {
			log.Printf("Error checking user ban status: %v", err)
			continue
		}
		if isBanned {
			comment.Content = "(user banned)"
		}
		modifyBannedUserComments(comment.Replies)
	}
}

func addCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	updateID := r.FormValue("update_id")
	parentID := r.FormValue("parent_id")
	content := r.FormValue("content")

	if content == "" {
		http.Error(w, "Comment content cannot be empty", http.StatusBadRequest)
		return
	}

	userID := getCurrentUserID(r)

	var err error
	var commentID int64
	if parentID == "" {
		err = db.QueryRow("INSERT INTO comments (update_id, user_id, content) VALUES ($1, $2, $3) RETURNING id",
			updateID, userID, content).Scan(&commentID)
	} else {
		err = db.QueryRow("INSERT INTO comments (update_id, user_id, parent_id, content) VALUES ($1, $2, $3, $4) RETURNING id",
			updateID, userID, parentID, content).Scan(&commentID)
	}

	if err != nil {
		log.Printf("Failed to add comment: %v", err)
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	// Redirect to the update page after successful comment submission
	http.Redirect(w, r, fmt.Sprintf("/update/%s", updateID), http.StatusSeeOther)
}

func getComments(updateID string) ([]*Comment, error) {
	rows, err := db.Query(`
        SELECT c.id, c.user_id, c.parent_id, c.content, c.created_at, 
               u.username, COALESCE(u.display_name, ''), COALESCE(u.profile_image_url, '')
        FROM comments c
        JOIN users u ON c.user_id = u.id
        WHERE c.update_id = $1
        ORDER BY c.created_at ASC
    `, updateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allComments []*Comment
	commentMap := make(map[int]*Comment)

	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.UserID, &comment.ParentID, &comment.Content, &comment.CreatedAt,
			&comment.Username, &comment.DisplayName, &comment.ProfileImageURL)
		if err != nil {
			return nil, err
		}
		commentMap[comment.ID] = &comment
		allComments = append(allComments, &comment)
	}

	var rootComments []*Comment
	for _, comment := range allComments {
		if !comment.ParentID.Valid {
			rootComments = append(rootComments, comment)
		} else {
			parentID := int(comment.ParentID.Int64)
			if parent, ok := commentMap[parentID]; ok {
				parent.Replies = append(parent.Replies, comment)
			}
		}
	}

	log.Printf("Root comments for update %s: %+v", updateID, rootComments)
	return rootComments, nil
}

func countTotalComments(comments []*Comment) int {
	total := len(comments)
	for _, comment := range comments {
		total += countTotalComments(comment.Replies)
	}
	return total
}

func generateRandomUsername() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const usernameLength = 8

	b := make([]byte, usernameLength)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return "user_" + string(b)
}
