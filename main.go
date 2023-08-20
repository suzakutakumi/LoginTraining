package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sqlx.DB
var mail Mail

func index(c *gin.Context) {
	if token, err := c.Cookie("token"); err != nil || token == "" {
		c.HTML(200, "index.html", gin.H{"login": false})
	} else {
		var user UserInfo
		if err := db.Get(&user, "SELECT users.id, users.name FROM users INNER JOIN token ON users.id=token.id WHERE token.uuid = ?", token); err != nil {
			log.Println(err)
			c.Status(http.StatusBadRequest)
			return
		}
		c.HTML(200, "index.html", gin.H{"login": true, "name": user.Name, "mail": user.Id})
	}
}

func loginPage(c *gin.Context) {
	if token, err := c.Cookie("token"); err != nil || token == "" {
		c.HTML(200, "login.html", gin.H{})
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}

func registerPage(c *gin.Context) {
	if token, err := c.Cookie("token"); err != nil || token == "" {
		c.HTML(200, "register.html", gin.H{})
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}

func checkNewID(c *gin.Context) {
	var id struct {
		Id string `json:"id"`
	}
	if err := c.ShouldBindJSON(&id); err != nil {
		log.Println("JSONのバインドがうまくできませんでした")
		c.Status(http.StatusBadRequest)
		return
	}

	var hoge interface{}
	if err := db.Get(hoge, "SELECT 1 FROM users WHERE id=$1", id.Id); err != nil {
		c.JSON(http.StatusOK, true)
	} else {
		c.JSON(http.StatusOK, false)
	}
}

func createAccount(c *gin.Context) {
	var user InputUserInfo
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Println("JSONのバインドがうまくできませんでした")
		c.Status(http.StatusBadRequest)
		return
	}

	var cnt int
	if err := db.Get(&cnt, "SELECT count(*) FROM users WHERE id = ?", user.Id); err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}
	if cnt > 0 {
		log.Println("既にユーザは存在しています")
		c.Status(http.StatusBadRequest)
		return
	}

	//bcrypt
	password_bytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	password := string(password_bytes)

	//uuid
	u, err := uuid.NewRandom()
	if err != nil {
		fmt.Println(err)
		return
	}
	uu := u.String()

	//dbにユーザ情報を保存
	_, err = db.NamedExec("INSERT INTO users VALUES(:id,:password,:name)", DBUser{user.Id, password, "Z"})
	if err != nil {
		log.Println("usersに挿入失敗")
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	//dbにアクティベーションフラグを保存
	_, err = db.NamedExec("INSERT INTO activate(id,uuid) VALUES(:id,:uuid)", Activate{user.Id, uu})
	if err != nil {
		log.Println("activateに挿入失敗")
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	//アクティベーションメールを送る
	msg := "以下のURLにアクセスしてアカウントがあなたのモノであることを示してください！\n"
	msg += "http://localhost:8080/api/user/signup/" + uu
	if err := mail.Send(user.Id, "Signup!", msg); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

func signup(c *gin.Context) {
	uuid := c.Param("id")
	log.Println(uuid)
	if _, err := db.Exec("DELETE FROM activate WHERE uuid = ?", uuid); err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError)
	}
	c.Redirect(http.StatusFound, "/login")
}

func login(c *gin.Context) {
	var user InputUserInfo
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Println("JSONのバインドがうまくできませんでした")
		c.Status(http.StatusBadRequest)
		return
	}

	var cnt int
	if err := db.Get(&cnt, "SELECT count(*) FROM users WHERE id = ?", user.Id); err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}
	if cnt == 0 {
		log.Println("ユーザが存在しません")
		c.Status(http.StatusBadRequest)
		return
	}

	//activateの状態を見る
	if err := db.Get(&cnt, "SELECT count(*) FROM activate WHERE id = ?", user.Id); err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}
	if cnt > 0 {
		log.Println("ユーザがアクティブではありません")
		c.Status(http.StatusBadRequest)
		return
	}

	//DBからパスワードを入手
	var hash string
	if err := db.Get(&hash, "SELECT password FROM users WHERE id = ?", user.Id); err != nil {
		log.Println("usersからの取得に失敗")
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	//認証
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Password)); err != nil {
		log.Println("パスワードが違います")
		log.Println(err)
		c.Status(http.StatusBadRequest)
		return
	}

	//tokenのuuid生成
	u, err := uuid.NewRandom()
	if err != nil {
		fmt.Println(err)
		return
	}
	uu := u.String()

	//token設定
	_, err = db.NamedExec("INSERT INTO token(id,uuid) VALUES(:id,:uuid)", Token{user.Id, uu})
	if err != nil {
		log.Println("tokenの挿入に失敗")
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.SetCookie("token", uu, 3600, "/", "localhost", true, true)
	c.Status(http.StatusOK)
}

func signout(c *gin.Context) {
	uuid, err := c.Request.Cookie("token")
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if _, err := db.Queryx("DELETE FROM token WHERE uuid = ?", uuid); err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError)
	}

	c.SetCookie("token", "", -1, "/", "localhost", true, true)
	c.Redirect(http.StatusFound, "/")
}

func main() {
	var err error
	db, err = sqlx.Connect("sqlite3", "db.sqlite3")
	if err != nil {
		log.Fatal("DBに接続できませんでした\n", err)
	}
	defer db.Close()

	if err := godotenv.Load(); err != nil {
		log.Fatal(".envファイルが読み込めませんでした")
	}
	host := os.Getenv("host")
	port, err := strconv.Atoi(os.Getenv("port"))
	if err != nil {
		log.Fatal("環境変数portが正しくありません")
	}
	sender := os.Getenv("mail")
	password := os.Getenv("password")
	log.Println(host, port, sender, password)
	mail = Mail{host, port, sender, password}

	router := gin.Default()
	router.LoadHTMLGlob("html/*")

	router.GET("/", index)
	router.GET("/login", loginPage)
	router.GET("/signup", registerPage)

	router.GET("api/user/signup/:id", signup)
	router.POST("/api/user", createAccount)
	router.POST("/api/user/login", login)
	router.POST("/api/user/signout", signout)

	router.Run()
}
