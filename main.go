package main

import (
	"database/sql"
	"errors" // pacote para lidar com erros.
	"fmt"
	"net/http" // pacote para interagir com funcionalidades HTTP.

	"github.com/gin-gonic/gin"      // framework Gin para criar APIs.
	_ "github.com/mattn/go-sqlite3" // importa o pacote SQLite para Go
)

var db *sql.DB

// estrutura do modelo de um livro.
type book struct {
	ID       string `json:"id"`       // ID no JSON.
	Title    string `json:"title"`    // Título no JSON.
	Author   string `json:"author"`   // Autor no JSON.
	Quantity int    `json:"quantity"` // Quantidade no JSON.
}

// função para inicializar o banco de dados SQLite.
func initDB() {
	var err error
	// cria ou abre o banco de dados SQLite
	db, err = sql.Open("sqlite3", "./books.db")
	if err != nil {
		fmt.Println("Erro ao abrir o banco de dados:", err)
		return
	}

	// cria a tabela de livros se não existir
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id TEXT PRIMARY KEY,
		title TEXT,
		author TEXT,
		quantity INTEGER
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		fmt.Println("Erro ao criar a tabela:", err)
		return
	}
}

// função para adicionar um novo livro à base de dados.
func createBook(c *gin.Context) {
	var newBook book
	if err := c.BindJSON(&newBook); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}

	// Verifica se o ID do livro já existe
	var existingBook book
	err := db.QueryRow("SELECT id FROM books WHERE id = ?", newBook.ID).Scan(&existingBook.ID)
	if err != nil && err != sql.ErrNoRows {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro ao verificar ID do livro: %v", err)})
		return
	}
	if existingBook.ID != "" {
		c.IndentedJSON(http.StatusConflict, gin.H{"message": "ID do livro já existe"})
		return
	}

	// Insere o livro no banco de dados
	_, err = db.Exec("INSERT INTO books(id, title, author, quantity) VALUES (?, ?, ?, ?)",
		newBook.ID, newBook.Title, newBook.Author, newBook.Quantity)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro ao adicionar livro: %v", err)})
		return
	}

	c.IndentedJSON(http.StatusCreated, newBook)
}

// função para retornar todos os livros.
func getBooks(c *gin.Context) {
	rows, err := db.Query("SELECT id, title, author, quantity FROM books")
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar livros"})
		return
	}
	defer rows.Close()

	var books []book
	for rows.Next() {
		var b book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Quantity); err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Erro ao ler os dados dos livros"})
			return
		}
		books = append(books, b)
	}

	c.IndentedJSON(http.StatusOK, books)
}

// função para buscar um livro pelo ID.
func getBookById(id string) (*book, error) {
	fmt.Printf("Buscando livro com ID: %s\n", id)
	row := db.QueryRow("SELECT id, title, author, quantity FROM books WHERE id = ?", id)

	var b book
	if err := row.Scan(&b.ID, &b.Title, &b.Author, &b.Quantity); err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Livro não encontrado")
			return nil, errors.New("livro não encontrado")
		}
		fmt.Printf("Erro ao buscar livro: %v\n", err)
		return nil, err
	}

	fmt.Printf("Livro encontrado: %+v\n", b)
	return &b, nil
}

// função para checkout de um livro.
func checkoutBook(c *gin.Context) {
	var request struct {
		ID string `json:"id"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}

	fmt.Printf("Requisição de checkout para o livro com ID: %s\n", request.ID)

	if request.ID == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "ID é necessário"})
		return
	}

	book, err := getBookById(request.ID)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Livro não encontrado"})
		return
	}

	if book.Quantity <= 0 {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Livro não disponível"})
		return
	}

	// atualiza a quantidade do livro
	_, err = db.Exec("UPDATE books SET quantity = quantity - 1 WHERE id = ?", request.ID)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Erro ao realizar checkout"})
		return
	}

	c.IndentedJSON(http.StatusOK, book)
}

// função para retorno de um livro.
func returnBook(c *gin.Context) {
	var request struct {
		ID string `json:"id"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Dados inválidos"})
		return
	}

	fmt.Printf("Requisição de retorno para o livro com ID: %s\n", request.ID)

	if request.ID == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "ID é necessário"})
		return
	}

	book, err := getBookById(request.ID)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Livro não encontrado"})
		return
	}

	// atualiza a quantidade do livro
	_, err = db.Exec("UPDATE books SET quantity = quantity + 1 WHERE id = ?", request.ID)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Erro ao retornar livro"})
		return
	}

	c.IndentedJSON(http.StatusOK, book)
}

// inicializa o servidor e registra as rotas.
func main() {
	initDB()
	defer db.Close()
	router := gin.Default() // cria um router padrão do Gin.

	// criar um livro
	router.POST("/books", createBook)

	// listar os livros
	router.GET("/books", getBooks)

	// buscar um livro pelo ID
	router.GET("/books/:id", func(c *gin.Context) {
		id := c.Param("id")
		book, err := getBookById(id)
		if err != nil {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Livro não encontrado"})
			return
		}
		c.IndentedJSON(http.StatusOK, book)
	})

	// realizar o checkout de um livro
	router.PATCH("/books/checkout", checkoutBook)

	// realizar o retorno de um livro
	router.PATCH("/books/return", returnBook)

	router.Run("localhost:8080")

	// para executar o servidor, execute o comando no terminal:

	// $env:PATH="C:\TDM-GCC-64\bin;$env:PATH"

	// $env:CGO_ENABLED=1
	// go build -o main.exe main.go

	// .\main.exe
}
