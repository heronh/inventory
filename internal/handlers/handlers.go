package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"inventory/internal/config"
	"inventory/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type App struct {
	DB       *gorm.DB
	Config   *config.Config
	views    *template.Template
	sessions map[string]uint
	mu       sync.RWMutex
}

type ViewData struct {
	Title     string
	Error     string
	Success   string
	User      *models.User
	Users     []models.User
	Tarefas   []models.Tarefas
	Roles     []models.Privilege
	Products  []models.Product
	Clients   []models.Client
	Suppliers []models.Supplier
	Entries   []models.Entry
	Sales     []models.Sale
	Entry     *models.Entry
	Sale      *models.Sale
	Client    *models.Client
	Supplier  *models.Supplier
	Product   *models.Product
}

func NewApp(db *gorm.DB, cfg *config.Config) (*App, error) {
	t, err := template.ParseGlob(filepath.Join("templates", "*.html"))
	if err != nil {
		return nil, err
	}

	return &App{
		DB:       db,
		Config:   cfg,
		views:    t,
		sessions: make(map[string]uint),
	}, nil
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", a.welcome)
	mux.HandleFunc("/login", a.login)
	mux.HandleFunc("/logout", a.logout)

	mux.HandleFunc("/user", a.requireAuth(a.userPage))
	mux.HandleFunc("/user/update", a.requireAuth(a.userUpdate))

	mux.HandleFunc("/newuser", a.requireRoles(a.newUserPage, "admin", "su"))
	mux.HandleFunc("/newuser/create", a.requireRoles(a.createUser, "admin", "su"))

	mux.HandleFunc("/inventory", a.requireAuth(a.inventoryPage))
	mux.HandleFunc("/tarefas", a.requireAuth(a.tarefasPage))
	mux.HandleFunc("/tarefas/create", a.requireAuth(a.createTarefa))
	mux.HandleFunc("/tarefas/update", a.requireAuth(a.updateTarefa))
	mux.HandleFunc("/tarefas/delete", a.requireAuth(a.deleteTarefa))

	mux.HandleFunc("/clients/new", a.requireAuth(a.newClientPage))
	mux.HandleFunc("/clients/create", a.requireAuth(a.createClient))
	mux.HandleFunc("/clients/edit", a.requireAuth(a.editClientPage))
	mux.HandleFunc("/clients/update", a.requireAuth(a.updateClient))

	mux.HandleFunc("/suppliers/new", a.requireAuth(a.newSupplierPage))
	mux.HandleFunc("/suppliers/create", a.requireAuth(a.createSupplier))
	mux.HandleFunc("/suppliers/edit", a.requireAuth(a.editSupplierPage))
	mux.HandleFunc("/suppliers/update", a.requireAuth(a.updateSupplier))

	mux.HandleFunc("/products/new", a.requireAuth(a.newProductPage))
	mux.HandleFunc("/products/create", a.requireAuth(a.createProduct))
	mux.HandleFunc("/products/edit", a.requireAuth(a.editProductPage))
	mux.HandleFunc("/products/update", a.requireAuth(a.updateProduct))

	mux.HandleFunc("/entries", a.requireAuth(a.entriesPage))
	mux.HandleFunc("/entries/create", a.requireAuth(a.createEntries))
	mux.HandleFunc("/entries/edit", a.requireAuth(a.editEntryPage))
	mux.HandleFunc("/entries/update", a.requireAuth(a.updateEntry))

	mux.HandleFunc("/sales", a.requireAuth(a.salesPage))
	mux.HandleFunc("/sales/create", a.requireAuth(a.createSales))
	mux.HandleFunc("/sales/edit", a.requireAuth(a.editSalePage))
	mux.HandleFunc("/sales/update", a.requireAuth(a.updateSale))

	return mux
}

func (a *App) welcome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	a.render(w, "welcome.html", ViewData{Title: "Bem-vindo"})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	var user models.User
	if err := a.DB.Preload("Privilege").Where("email = ?", email).First(&user).Error; err != nil {
		a.render(w, "welcome.html", ViewData{Title: "Bem-vindo", Error: "credenciais inválidas"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		a.render(w, "welcome.html", ViewData{Title: "Bem-vindo", Error: "credenciais inválidas"})
		return
	}

	token := uuid.NewString()
	a.mu.Lock()
	a.sessions[token] = user.ID
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "inventory_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600 * 8,
	})

	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("inventory_session")
	if err == nil {
		a.mu.Lock()
		delete(a.sessions, cookie.Value)
		a.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "inventory_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) userPage(w http.ResponseWriter, r *http.Request) {
	user, err := a.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	a.render(w, "user.html", ViewData{Title: "Usuário", User: user})
}

func (a *App) userUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}

	user, err := a.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	password := strings.TrimSpace(r.FormValue("password"))
	if name != "" {
		user.Name = name
	}
	if password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			a.render(w, "user.html", ViewData{Title: "Usuário", User: user, Error: "não foi possível gerar hash da senha"})
			return
		}
		user.Password = string(hashed)
	}

	if err = a.DB.Save(user).Error; err != nil {
		a.render(w, "user.html", ViewData{Title: "Usuário", User: user, Error: "não foi possível atualizar o usuário"})
		return
	}

	_ = a.createLog(user.ID, "atualizou o próprio perfil")
	a.render(w, "user.html", ViewData{Title: "Usuário", User: user, Success: "perfil atualizado"})
}

func (a *App) newUserPage(w http.ResponseWriter, r *http.Request) {
	roles := []models.Privilege{}
	_ = a.DB.Order("description asc").Find(&roles).Error

	user, _ := a.currentUser(r)
	a.render(w, "newuser.html", ViewData{Title: "Novo Usuário", User: user, Roles: roles})
}

func (a *App) createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/newuser", http.StatusSeeOther)
		return
	}

	current, _ := a.currentUser(r)
	roles := []models.Privilege{}
	_ = a.DB.Order("description asc").Find(&roles).Error

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	roleID, err := strconv.ParseUint(r.FormValue("privilege_id"), 10, 64)
	if err != nil || name == "" || email == "" || password == "" {
		a.render(w, "newuser.html", ViewData{Title: "Novo Usuário", User: current, Roles: roles, Error: "dados do formulário inválidos"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		a.render(w, "newuser.html", ViewData{Title: "Novo Usuário", User: current, Roles: roles, Error: "não foi possível gerar hash da senha"})
		return
	}

	newUser := models.User{
		Name:        name,
		Email:       email,
		Password:    string(hashed),
		PrivilegeID: uint(roleID),
		Phone:       phone,
	}

	if err = a.DB.Create(&newUser).Error; err != nil {
		a.render(w, "newuser.html", ViewData{Title: "Novo Usuário", User: current, Roles: roles, Error: "não foi possível criar usuário"})
		return
	}

	_ = a.createLog(current.ID, fmt.Sprintf("criou usuário %s", email))
	a.render(w, "newuser.html", ViewData{Title: "Novo Usuário", User: current, Roles: roles, Success: "usuário criado"})
}

func (a *App) inventoryPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	products := []models.Product{}
	_ = a.DB.Order("name asc").Find(&products).Error
	a.render(w, "inventory.html", ViewData{Title: "Estoque", User: user, Products: products})
}

func (a *App) tarefasPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	tarefas := []models.Tarefas{}
	_ = a.DB.Preload("User").Order("created_at desc").Find(&tarefas).Error
	errMsg := strings.TrimSpace(r.URL.Query().Get("error"))
	successMsg := strings.TrimSpace(r.URL.Query().Get("success"))

	a.render(w, "tarefas.html", ViewData{Title: "Tarefas", User: user, Tarefas: tarefas, Error: errMsg, Success: successMsg})
}

func (a *App) createTarefa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tarefas", http.StatusSeeOther)
		return
	}

	user, err := a.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	status := normalizeTaskStatus(r.FormValue("status"))
	if description == "" {
		http.Redirect(w, r, "/tarefas?error=descrição+é+obrigatória", http.StatusSeeOther)
		return
	}

	tarefa := models.Tarefas{
		Description: description,
		Status:      status,
		UserID:      user.ID,
	}

	if err = a.DB.Create(&tarefa).Error; err != nil {
		http.Redirect(w, r, "/tarefas?error=não+foi+possível+criar+tarefa", http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("criou tarefa %d", tarefa.ID))
	http.Redirect(w, r, "/tarefas?success=tarefa+criada", http.StatusSeeOther)
}

func (a *App) updateTarefa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tarefas", http.StatusSeeOther)
		return
	}

	user, err := a.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.Redirect(w, r, "/tarefas?error=tarefa+inválida", http.StatusSeeOther)
		return
	}

	var tarefa models.Tarefas
	if err = a.DB.First(&tarefa, id).Error; err != nil {
		http.Redirect(w, r, "/tarefas?error=tarefa+não+encontrada", http.StatusSeeOther)
		return
	}

	if tarefa.UserID != user.ID && !isPrivilegedUser(user) {
		http.Redirect(w, r, "/tarefas?error=sem+permissão+para+alterar+esta+tarefa", http.StatusSeeOther)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	if description == "" {
		http.Redirect(w, r, "/tarefas?error=descrição+é+obrigatória", http.StatusSeeOther)
		return
	}

	tarefa.Description = description
	tarefa.Status = normalizeTaskStatus(r.FormValue("status"))

	if err = a.DB.Save(&tarefa).Error; err != nil {
		http.Redirect(w, r, "/tarefas?error=não+foi+possível+atualizar+tarefa", http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("atualizou tarefa %d", tarefa.ID))
	http.Redirect(w, r, "/tarefas?success=tarefa+atualizada", http.StatusSeeOther)
}

func (a *App) deleteTarefa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tarefas", http.StatusSeeOther)
		return
	}

	user, err := a.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.Redirect(w, r, "/tarefas?error=tarefa+inválida", http.StatusSeeOther)
		return
	}

	var tarefa models.Tarefas
	if err = a.DB.First(&tarefa, id).Error; err != nil {
		http.Redirect(w, r, "/tarefas?error=tarefa+não+encontrada", http.StatusSeeOther)
		return
	}

	if tarefa.UserID != user.ID && !isPrivilegedUser(user) {
		http.Redirect(w, r, "/tarefas?error=sem+permissão+para+excluir+esta+tarefa", http.StatusSeeOther)
		return
	}

	if err = a.DB.Delete(&tarefa).Error; err != nil {
		http.Redirect(w, r, "/tarefas?error=não+foi+possível+excluir+tarefa", http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("excluiu tarefa %d", id))
	http.Redirect(w, r, "/tarefas?success=tarefa+excluída", http.StatusSeeOther)
}

func (a *App) newClientPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	a.render(w, "newclient.html", ViewData{Title: "Novo Cliente", User: user})
}

func (a *App) createClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/clients/new", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	client := models.Client{
		Name:         strings.TrimSpace(r.FormValue("name")),
		Phone:        strings.TrimSpace(r.FormValue("phone")),
		TaxID:        strings.TrimSpace(r.FormValue("tax_id")),
		ZIPCode:      strings.TrimSpace(r.FormValue("zip_code")),
		Street:       strings.TrimSpace(r.FormValue("street")),
		Number:       strings.TrimSpace(r.FormValue("number")),
		Complement:   strings.TrimSpace(r.FormValue("complement")),
		Neighborhood: strings.TrimSpace(r.FormValue("neighborhood")),
		City:         strings.TrimSpace(r.FormValue("city")),
		State:        strings.TrimSpace(r.FormValue("state")),
	}

	if client.Name == "" {
		a.render(w, "newclient.html", ViewData{Title: "Novo Cliente", User: user, Error: "nome é obrigatório"})
		return
	}

	if err := a.DB.Create(&client).Error; err != nil {
		a.render(w, "newclient.html", ViewData{Title: "Novo Cliente", User: user, Error: "não foi possível criar cliente"})
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("criou cliente %s", client.Name))
	a.render(w, "newclient.html", ViewData{Title: "Novo Cliente", User: user, Success: "cliente criado"})
}

func (a *App) editClientPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	clients := []models.Client{}
	_ = a.DB.Order("name asc").Find(&clients).Error

	id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	var selected *models.Client
	if id > 0 {
		var c models.Client
		if err := a.DB.First(&c, id).Error; err == nil {
			selected = &c
		}
	}

	a.render(w, "editclient.html", ViewData{Title: "Editar Cliente", User: user, Clients: clients, Client: selected})
}

func (a *App) updateClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/clients/edit", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.Redirect(w, r, "/clients/edit", http.StatusSeeOther)
		return
	}

	var client models.Client
	if err = a.DB.First(&client, id).Error; err != nil {
		http.Redirect(w, r, "/clients/edit", http.StatusSeeOther)
		return
	}

	client.Name = strings.TrimSpace(r.FormValue("name"))
	client.Phone = strings.TrimSpace(r.FormValue("phone"))
	client.TaxID = strings.TrimSpace(r.FormValue("tax_id"))
	client.ZIPCode = strings.TrimSpace(r.FormValue("zip_code"))
	client.Street = strings.TrimSpace(r.FormValue("street"))
	client.Number = strings.TrimSpace(r.FormValue("number"))
	client.Complement = strings.TrimSpace(r.FormValue("complement"))
	client.Neighborhood = strings.TrimSpace(r.FormValue("neighborhood"))
	client.City = strings.TrimSpace(r.FormValue("city"))
	client.State = strings.TrimSpace(r.FormValue("state"))

	if err = a.DB.Save(&client).Error; err != nil {
		http.Redirect(w, r, "/clients/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("atualizou cliente %d", id))
	http.Redirect(w, r, "/clients/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
}

func (a *App) newSupplierPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	a.render(w, "newsupplier.html", ViewData{Title: "Novo Fornecedor", User: user})
}

func (a *App) createSupplier(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/suppliers/new", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	supplier := models.Supplier{
		Name:         strings.TrimSpace(r.FormValue("name")),
		TradeName:    strings.TrimSpace(r.FormValue("trade_name")),
		TaxID:        strings.TrimSpace(r.FormValue("tax_id")),
		ZIPCode:      strings.TrimSpace(r.FormValue("zip_code")),
		Street:       strings.TrimSpace(r.FormValue("street")),
		Number:       strings.TrimSpace(r.FormValue("number")),
		Complement:   strings.TrimSpace(r.FormValue("complement")),
		Neighborhood: strings.TrimSpace(r.FormValue("neighborhood")),
		City:         strings.TrimSpace(r.FormValue("city")),
		State:        strings.TrimSpace(r.FormValue("state")),
	}

	if supplier.Name == "" {
		a.render(w, "newsupplier.html", ViewData{Title: "Novo Fornecedor", User: user, Error: "nome é obrigatório"})
		return
	}

	if err := a.DB.Create(&supplier).Error; err != nil {
		a.render(w, "newsupplier.html", ViewData{Title: "Novo Fornecedor", User: user, Error: "não foi possível criar fornecedor"})
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("criou fornecedor %s", supplier.Name))
	a.render(w, "newsupplier.html", ViewData{Title: "Novo Fornecedor", User: user, Success: "fornecedor criado"})
}

func (a *App) editSupplierPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	suppliers := []models.Supplier{}
	_ = a.DB.Order("name asc").Find(&suppliers).Error

	id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	var selected *models.Supplier
	if id > 0 {
		var supplier models.Supplier
		if err := a.DB.First(&supplier, id).Error; err == nil {
			selected = &supplier
		}
	}

	a.render(w, "editsupplier.html", ViewData{Title: "Editar Fornecedor", User: user, Suppliers: suppliers, Supplier: selected})
}

func (a *App) updateSupplier(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/suppliers/edit", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.Redirect(w, r, "/suppliers/edit", http.StatusSeeOther)
		return
	}

	var supplier models.Supplier
	if err = a.DB.First(&supplier, id).Error; err != nil {
		http.Redirect(w, r, "/suppliers/edit", http.StatusSeeOther)
		return
	}

	supplier.Name = strings.TrimSpace(r.FormValue("name"))
	supplier.TradeName = strings.TrimSpace(r.FormValue("trade_name"))
	supplier.TaxID = strings.TrimSpace(r.FormValue("tax_id"))
	supplier.ZIPCode = strings.TrimSpace(r.FormValue("zip_code"))
	supplier.Street = strings.TrimSpace(r.FormValue("street"))
	supplier.Number = strings.TrimSpace(r.FormValue("number"))
	supplier.Complement = strings.TrimSpace(r.FormValue("complement"))
	supplier.Neighborhood = strings.TrimSpace(r.FormValue("neighborhood"))
	supplier.City = strings.TrimSpace(r.FormValue("city"))
	supplier.State = strings.TrimSpace(r.FormValue("state"))

	if err = a.DB.Save(&supplier).Error; err != nil {
		http.Redirect(w, r, "/suppliers/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("atualizou fornecedor %d", id))
	http.Redirect(w, r, "/suppliers/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
}

func (a *App) newProductPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	a.render(w, "newproduct.html", ViewData{Title: "Novo Produto", User: user})
}

func (a *App) createProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/products/new", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	quantity, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
	minStock, _ := strconv.ParseFloat(r.FormValue("minimum_stock"), 64)

	product := models.Product{
		Name:         strings.TrimSpace(r.FormValue("name")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		Code:         strings.TrimSpace(r.FormValue("code")),
		Price:        price,
		Quantity:     quantity,
		Unit:         strings.TrimSpace(r.FormValue("unit")),
		MinimumStock: minStock,
	}

	if product.Name == "" || product.Code == "" || product.Unit == "" {
		a.render(w, "newproduct.html", ViewData{Title: "Novo Produto", User: user, Error: "nome, código e unidade são obrigatórios"})
		return
	}

	if err := a.DB.Create(&product).Error; err != nil {
		a.render(w, "newproduct.html", ViewData{Title: "Novo Produto", User: user, Error: "não foi possível criar produto"})
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("criou produto %s", product.Code))
	a.render(w, "newproduct.html", ViewData{Title: "Novo Produto", User: user, Success: "produto criado"})
}

func (a *App) editProductPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	products := []models.Product{}
	_ = a.DB.Order("name asc").Find(&products).Error

	id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	var selected *models.Product
	if id > 0 {
		var product models.Product
		if err := a.DB.First(&product, id).Error; err == nil {
			selected = &product
		}
	}

	a.render(w, "editproduct.html", ViewData{Title: "Editar Produto", User: user, Products: products, Product: selected})
}

func (a *App) updateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/products/edit", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.Redirect(w, r, "/products/edit", http.StatusSeeOther)
		return
	}

	var product models.Product
	if err = a.DB.First(&product, id).Error; err != nil {
		http.Redirect(w, r, "/products/edit", http.StatusSeeOther)
		return
	}

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	quantity, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
	minStock, _ := strconv.ParseFloat(r.FormValue("minimum_stock"), 64)

	product.Name = strings.TrimSpace(r.FormValue("name"))
	product.Description = strings.TrimSpace(r.FormValue("description"))
	product.Code = strings.TrimSpace(r.FormValue("code"))
	product.Price = price
	product.Quantity = quantity
	product.Unit = strings.TrimSpace(r.FormValue("unit"))
	product.MinimumStock = minStock

	if err = a.DB.Save(&product).Error; err != nil {
		http.Redirect(w, r, "/products/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("atualizou produto %d", id))
	http.Redirect(w, r, "/products/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
}

func (a *App) entriesPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	products := []models.Product{}
	suppliers := []models.Supplier{}
	entries := []models.Entry{}
	_ = a.DB.Order("name asc").Find(&products).Error
	_ = a.DB.Order("name asc").Find(&suppliers).Error
	_ = a.DB.Preload("Product").Preload("Supplier").Order("created_at desc").Limit(50).Find(&entries).Error

	a.render(w, "entries.html", ViewData{
		Title:     "Entradas",
		User:      user,
		Products:  products,
		Suppliers: suppliers,
		Entries:   entries,
	})
}

func (a *App) createEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/entries", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/entries", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	productIDs := r.Form["product_id"]
	supplierIDs := r.Form["supplier_id"]
	quantities := r.Form["quantity"]
	prices := r.Form["price"]
	observations := r.Form["observation"]

	if len(productIDs) == 0 || len(productIDs) != len(quantities) || len(productIDs) != len(prices) || len(productIDs) != len(supplierIDs) {
		http.Redirect(w, r, "/entries", http.StatusSeeOther)
		return
	}

	err := a.DB.Transaction(func(tx *gorm.DB) error {
		hasValidLine := false
		for i := range productIDs {
			pidRaw := strings.TrimSpace(productIDs[i])
			sidRaw := strings.TrimSpace(supplierIDs[i])
			qtyRaw := strings.TrimSpace(quantities[i])
			priceRaw := strings.TrimSpace(prices[i])

			if pidRaw == "" && sidRaw == "" && qtyRaw == "" && priceRaw == "" {
				continue
			}
			hasValidLine = true

			pid, err := strconv.ParseUint(productIDs[i], 10, 64)
			if err != nil {
				return err
			}
			sid, err := strconv.ParseUint(supplierIDs[i], 10, 64)
			if err != nil {
				return err
			}
			qty, err := strconv.ParseFloat(quantities[i], 64)
			if err != nil || qty <= 0 {
				return errors.New("quantidade inválida")
			}
			price, err := strconv.ParseFloat(prices[i], 64)
			if err != nil || price < 0 {
				return errors.New("preço inválido")
			}

			entry := models.Entry{
				UserID:      user.ID,
				Observation: at(observations, i),
				ProductID:   uint(pid),
				Quantity:    qty,
				SupplierID:  uint(sid),
				Price:       price,
			}
			if err = tx.Create(&entry).Error; err != nil {
				return err
			}

			if err = tx.Model(&models.Product{}).Where("id = ?", pid).Update("quantity", gorm.Expr("quantity + ?", qty)).Error; err != nil {
				return err
			}
		}
		if !hasValidLine {
			return errors.New("nenhuma linha de entrada válida")
		}

		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/entries", http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, "registrou entradas de estoque")
	http.Redirect(w, r, "/entries", http.StatusSeeOther)
}

func (a *App) editEntryPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	entries := []models.Entry{}
	_ = a.DB.Preload("Product").Preload("Supplier").Order("created_at desc").Limit(100).Find(&entries).Error

	id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	var selected *models.Entry
	if id > 0 {
		var entry models.Entry
		if err := a.DB.Preload("Product").Preload("Supplier").First(&entry, id).Error; err == nil {
			selected = &entry
		}
	}

	a.render(w, "editentries.html", ViewData{Title: "Editar Entradas", User: user, Entries: entries, Entry: selected})
}

func (a *App) updateEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/entries/edit", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.Redirect(w, r, "/entries/edit", http.StatusSeeOther)
		return
	}

	newQty, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
	newPrice, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	newObs := strings.TrimSpace(r.FormValue("observation"))
	if newQty <= 0 || newPrice < 0 {
		http.Redirect(w, r, "/entries/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
		return
	}

	err = a.DB.Transaction(func(tx *gorm.DB) error {
		var entry models.Entry
		if err := tx.First(&entry, id).Error; err != nil {
			return err
		}

		delta := newQty - entry.Quantity
		if err := tx.Model(&models.Product{}).Where("id = ?", entry.ProductID).Update("quantity", gorm.Expr("quantity + ?", delta)).Error; err != nil {
			return err
		}

		entry.Quantity = newQty
		entry.Price = newPrice
		entry.Observation = newObs
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/entries/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("atualizou entrada %d", id))
	http.Redirect(w, r, "/entries/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
}

func (a *App) salesPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	products := []models.Product{}
	clients := []models.Client{}
	sales := []models.Sale{}
	_ = a.DB.Order("name asc").Find(&products).Error
	_ = a.DB.Order("name asc").Find(&clients).Error
	_ = a.DB.Preload("Product").Preload("Client").Order("created_at desc").Limit(50).Find(&sales).Error
	errMsg := strings.TrimSpace(r.URL.Query().Get("error"))
	successMsg := strings.TrimSpace(r.URL.Query().Get("success"))

	a.render(w, "sales.html", ViewData{
		Title:    "Vendas",
		User:     user,
		Products: products,
		Clients:  clients,
		Sales:    sales,
		Error:    errMsg,
		Success:  successMsg,
	})
}

func (a *App) createSales(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/sales", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/sales?error=falha+ao+processar+formulário", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	productIDs := r.Form["product_id"]
	quantities := r.Form["quantity"]
	prices := r.Form["price"]
	observations := r.Form["observation"]
	clientID, err := strconv.ParseUint(r.FormValue("client_id"), 10, 64)
	if err != nil || clientID == 0 {
		http.Redirect(w, r, "/sales?error=cliente+é+obrigatório", http.StatusSeeOther)
		return
	}

	if len(productIDs) == 0 {
		if singleProduct := strings.TrimSpace(r.FormValue("product_id")); singleProduct != "" {
			productIDs = []string{singleProduct}
			quantities = []string{r.FormValue("quantity")}
			prices = []string{r.FormValue("price")}
			observations = []string{r.FormValue("observation")}
		}
	}

	if len(productIDs) == 0 {
		http.Redirect(w, r, "/sales?error=produto+é+obrigatório", http.StatusSeeOther)
		return
	}

	err = a.DB.Transaction(func(tx *gorm.DB) error {
		hasValidLine := false
		for i := range productIDs {
			pidRaw := strings.TrimSpace(productIDs[i])
			qtyRaw := strings.TrimSpace(at(quantities, i))
			priceRaw := strings.TrimSpace(at(prices, i))

			if pidRaw == "" && qtyRaw == "" && priceRaw == "" {
				continue
			}
			if pidRaw == "" || qtyRaw == "" || priceRaw == "" {
				return errors.New("produto, quantidade e preço são obrigatórios")
			}
			hasValidLine = true

			pid, err := strconv.ParseUint(pidRaw, 10, 64)
			if err != nil {
				return err
			}
			qty, err := parseDecimalInput(qtyRaw)
			if err != nil || qty <= 0 {
				return errors.New("quantidade inválida")
			}
			price, err := parseDecimalInput(priceRaw)
			if err != nil || price < 0 {
				return errors.New("preço inválido")
			}

			var product models.Product
			if err = tx.First(&product, pid).Error; err != nil {
				return err
			}
			if product.Quantity < qty {
				return errors.New("estoque insuficiente")
			}

			sale := models.Sale{
				UserID:      user.ID,
				Observation: at(observations, i),
				ProductID:   uint(pid),
				Quantity:    qty,
				ClientID:    uint(clientID),
				Price:       price,
			}
			if err = tx.Create(&sale).Error; err != nil {
				return err
			}

			if err = tx.Model(&models.Product{}).Where("id = ?", pid).Update("quantity", gorm.Expr("quantity - ?", qty)).Error; err != nil {
				return err
			}
		}
		if !hasValidLine {
			return errors.New("nenhuma linha de venda válida")
		}

		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/sales?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, "registrou vendas")
	http.Redirect(w, r, "/sales?success=venda+registrada", http.StatusSeeOther)
}

func (a *App) editSalePage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.currentUser(r)
	sales := []models.Sale{}
	_ = a.DB.Preload("Product").Preload("Client").Order("created_at desc").Limit(100).Find(&sales).Error

	id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	var selected *models.Sale
	if id > 0 {
		var sale models.Sale
		if err := a.DB.Preload("Product").Preload("Client").First(&sale, id).Error; err == nil {
			selected = &sale
		}
	}

	a.render(w, "editsales.html", ViewData{Title: "Editar Vendas", User: user, Sales: sales, Sale: selected})
}

func (a *App) updateSale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/sales/edit", http.StatusSeeOther)
		return
	}

	user, _ := a.currentUser(r)
	id, err := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.Redirect(w, r, "/sales/edit", http.StatusSeeOther)
		return
	}

	newQty, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
	newPrice, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	newObs := strings.TrimSpace(r.FormValue("observation"))
	if newQty <= 0 || newPrice < 0 {
		http.Redirect(w, r, "/sales/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
		return
	}

	err = a.DB.Transaction(func(tx *gorm.DB) error {
		var sale models.Sale
		if err := tx.First(&sale, id).Error; err != nil {
			return err
		}

		var product models.Product
		if err := tx.First(&product, sale.ProductID).Error; err != nil {
			return err
		}

		delta := newQty - sale.Quantity
		if delta > 0 && product.Quantity < delta {
			return errors.New("estoque insuficiente")
		}

		if err := tx.Model(&models.Product{}).Where("id = ?", sale.ProductID).Update("quantity", gorm.Expr("quantity - ?", delta)).Error; err != nil {
			return err
		}

		sale.Quantity = newQty
		sale.Price = newPrice
		sale.Observation = newObs
		if err := tx.Save(&sale).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/sales/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
		return
	}

	_ = a.createLog(user.ID, fmt.Sprintf("atualizou venda %d", id))
	http.Redirect(w, r, "/sales/edit?id="+strconv.FormatUint(id, 10), http.StatusSeeOther)
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := a.currentUser(r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *App) requireRoles(next http.HandlerFunc, allowed ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := a.currentUser(r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		for _, role := range allowed {
			if strings.EqualFold(user.Privilege.Description, role) {
				next(w, r)
				return
			}
		}

		http.Error(w, "proibido", http.StatusForbidden)
	}
}

func (a *App) currentUser(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("inventory_session")
	if err != nil {
		return nil, err
	}

	a.mu.RLock()
	uid, ok := a.sessions[cookie.Value]
	a.mu.RUnlock()
	if !ok {
		return nil, errors.New("sessão inválida")
	}

	var user models.User
	if err = a.DB.Preload("Privilege").First(&user, uid).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (a *App) render(w http.ResponseWriter, view string, data ViewData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.views.ExecuteTemplate(w, view, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) createLog(userID uint, message string) error {
	entry := models.Log{Description: message, UserID: userID}
	return a.DB.Create(&entry).Error
}

func at(values []string, idx int) string {
	if idx >= 0 && idx < len(values) {
		return strings.TrimSpace(values[idx])
	}
	return ""
}

func parseDecimalInput(value string) (float64, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	return strconv.ParseFloat(normalized, 64)
}

func normalizeTaskStatus(value string) string {
	status := strings.ToLower(strings.TrimSpace(value))
	switch status {
	case "pendente", "em andamento", "concluída":
		return status
	default:
		return "pendente"
	}
}

func isPrivilegedUser(user *models.User) bool {
	if user == nil {
		return false
	}
	role := strings.TrimSpace(strings.ToLower(user.Privilege.Description))
	return role == "admin" || role == "su"
}
