# Sistema de Controle de Estoque

Aplicação completa de gestão de estoque com backend em Go, banco PostgreSQL, execução com Docker e páginas HTML no frontend.

## Stack

- Backend Go (`net/http` + `gorm`)
- Banco de dados PostgreSQL
- Docker e Docker Compose
- Templates HTML

## Ambiente

A raiz do projeto contém um arquivo `.env` com:

- Variáveis de conexão do banco e da aplicação
- Chave secreta da aplicação
- Variáveis de seed de perfis (`su`, `admin`, `user`)
- Variáveis de seed do usuário inicial

### Conexão a Bancos de Dados Externos (ex: Google Cloud SQL)

Você pode desativar a rotina que inicializa os conteineres Docker do PostgreSQL automaticamente adicionando a seguinte flag ao seu `.env`:

```env
DISABLE_DB_BOOTSTRAP=true
```

Para bancos da nuvem que exigem SSL obrigatório, você também pode adicionar e gerenciar o `DB_SSLMODE` (ex: `require`, `disable`):

```env
DB_SSLMODE=require
```

## Compilação e Execução Manual

Para baixar as dependências e compilar o binário Go da aplicação, execute:

```bash
go mod download
go build -o inventory ./cmd/server
```

Após compilado, o executável `inventory` será criado na raiz do projeto. Para rodar a aplicação usando esse binário:

```bash
./inventory
```

> **Aviso:** O serviço do banco de dados (PostgreSQL) precisará estar rodando com os dados de conexão listados no seu arquivo `.env` para a aplicação funcionar perfeitamente.

Você também pode utilizar o modo em desenvolvimento nativo do Go diretamente sem compilar:

```bash
go run ./cmd/server
```

## Execução via Make (Recomendado na máquina local)

1. Construa a imagem customizada do PostgreSQL:

```bash
make postgres-image
```

2. Inicie banco e aplicação:

```bash
make start
```

Ou execute o servidor diretamente (ele verifica/inicia o container do banco quando necessário):

```bash
make run
```

## Execução com Docker Compose

```bash
make up
```

Para parar:

```bash
make down
```

## Acesso Padrão

Usuário semeado a partir do `.env`:

- Email: `heronhurpia@gmail.com`
- Senha: `123mudar`

## Páginas Principais

- `/` → tela inicial/login
- `/user` → edição do perfil do usuário
- `/newuser` → criação de usuários (`admin` ou `su`)
- `/inventory` → produtos e quantidade em estoque
- `/clients/new`, `/clients/edit`
- `/products/new`, `/products/edit`
- `/suppliers/new`, `/suppliers/edit`
- `/entries`, `/entries/edit`
- `/sales`, `/sales/edit`

## Observações

- As migrações executam automaticamente na inicialização.
- Os perfis e o usuário inicial são semeados automaticamente na inicialização.
- Entradas aumentam o estoque; vendas diminuem o estoque.