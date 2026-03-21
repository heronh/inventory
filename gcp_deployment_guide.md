# Guia de Deploy no Google Cloud (Cloud Run + Cloud SQL)

Este guia explica o passo a passo para colocar sua aplicação de controle de estoque Go no ar utilizando os serviços gerenciados do Google Cloud: **Cloud Run** para a aplicação e **Cloud SQL** para o banco de dados PostgreSQL.

## Pré-requisitos
1. Ter uma conta no [Google Cloud Platform (GCP)](https://console.cloud.google.com/).
2. Ter instalado o [Google Cloud CLI (`gcloud`)](https://cloud.google.com/sdk/docs/install) em sua máquina.
3. Autenticar no terminal rodando:
   ```bash
   gcloud auth login
   gcloud config set project [SEU_PROJECT_ID] # Ex: inventory-beautybrasil-project
   ```

---

## 1. Ativar as APIs Necessárias no GCP
No terminal, certifique-se de habilitar as APIs essenciais para executar a aplicação:
```bash
gcloud services enable run.googleapis.com \
    sqladmin.googleapis.com \
    artifactregistry.googleapis.com \
    cloudbuild.googleapis.com
```

---

## 2. Configurar o Banco de Dados (Cloud SQL para PostgreSQL)

Sua aplicação utiliza um banco PostgreSQL. No GCP, a melhor forma de hospedá-lo é no Cloud SQL.

1. Crie uma instância de banco de dados (pode levar alguns minutos):
   ```bash
   gcloud sql instances create inventory-db-instance \
       --database-version=POSTGRES_15 \
       --cpu=1 --memory=3840MB \
       --region=southamerica-east1 \
       --root-password="[SUA_SENHA_FORTE]"
   ```
   *(Substitua `[SUA_SENHA_FORTE]` pela senha que está no seu [.env-cloud](file:///Users/heronhurpia/Sites/inventory/.env-cloud))*

2. Crie o banco de dados chamado [inventorydb](file:///Users/heronhurpia/Sites/inventory/inventorydb) (ou o nome definido no seu `DB_NAME`):
   ```bash
   gcloud sql databases create inventorydb --instance=inventory-db-instance
   ```

3. Obtenha o nome da conexão da instância (Instance Connection Name):
   ```bash
   gcloud sql instances describe inventory-db-instance --format="value(connectionName)"
   ```
   *Guarde esse valor, que terá o formato: `PROJECT_ID:REGION:INSTANCE_ID`. Ele será usado durante a configuração da aplicação.*

---

## 3. Preparar o Docker do Artefato (Artifact Registry)

Antes de enviar a aplicação para o Cloud Run, precisamos criar um repositório para guardar a imagem Docker que foi construída na sua máquina.

1. Crie o repositório no Artifact Registry:
   ```bash
   gcloud artifacts repositories create inventory-repo \
       --repository-format=docker \
       --location=southamerica-east1
   ```

2. Autentique o Docker local com o Artifact Registry do Google Cloud:
   ```bash
   gcloud auth configure-docker southamerica-east1-docker.pkg.dev
   ```

---

## 4. Construir e Fazer o Push da Imagem da Aplicação

1. Certifique-se de estar na raiz do projeto (onde está o seu [Dockerfile](file:///Users/heronhurpia/Sites/inventory/Dockerfile)).
2. Faça o Build da imagem, marcando-a para o seu Artifact Registry local:
   ```bash
   docker build -t southamerica-east1-docker.pkg.dev/[PROJECT_ID]/inventory-repo/inventory-app:latest .
   ```
   *(Substitua `[PROJECT_ID]` pelo ID do seu projeto Google Cloud).*

3. Faça o push (envio) da imagem para o repositório na nuvem:
   ```bash
   docker push southamerica-east1-docker.pkg.dev/[PROJECT_ID]/inventory-repo/inventory-app:latest
   ```

---

## 5. Fazer o Deploy para o Google Cloud Run

Para que o Cloud Run consiga se comunicar com o banco de dados via uma conexão segura (sem expor o banco à internet pública diretamente), vamos fazer o deploy vinculando ao Cloud SQL.

**Variáveis de Ambiente Importantes ([.env](file:///Users/heronhurpia/Sites/inventory/.env)):**
Como consta no seu [README.md](file:///Users/heronhurpia/Sites/inventory/README.md), é importante usarmos `DISABLE_DB_BOOTSTRAP=true` para que a aplicação não tente construir os containers Docker internamente, já que eles não funcionam nem são necessários no Cloud Run.

Execute o comando de deploy. *(Substitua `[PROJECT_ID]` pelo seu ID e utilize o formato correto dos parâmetros)*:

```bash
gcloud run deploy inventory-service \
    --image southamerica-east1-docker.pkg.dev/inventory-488915/inventory-repo/inventory-app:latest \
    --region southamerica-east1 \
    --allow-unauthenticated \
    --set-env-vars "DISABLE_DB_BOOTSTRAP=true" \
    --set-env-vars "DB_HOST=/cloudsql/inventory-488915:southamerica-east1:inventory-db-instance" \
    --set-env-vars "DB_USER=postgres" \
    --set-env-vars 'DB_PASSWORD=ZP"z=+"I9X_(Qvj8' \
    --set-env-vars "DB_NAME=inventorydb" \
    --set-env-vars 'SECRET_KEY=hhheew566#4SW2!HJHgr' \
    --add-cloudsql-instances inventory-488915:southamerica-east1:inventory-db-instance
```

> **Atenção:** Em ambientes de produção do Google Cloud usando a biblioteca `net/http` ou via GORM, o `DB_HOST` pode precisar da sintaxe em forma de unix socket se acessado via Cloud SQL Connector, por exemplo: `host=/cloudsql/PROJECT_ID:REGION:INSTANCE_ID`. No seu código, garanta que essa variável passe sem problemas diretamente para a string de conexão (DSN) do GORM.

---

## 6. Acessar a Aplicação

Após rodar o comando acima, o log emitirá uma mensagem de sucesso no final com a URL gerada (ex: `https://inventory-service-xxxxxxxx-rj.a.run.app`).

1. Acesse o URL gerada usando o seu navegador.
2. Como você possui rotinas no Go que rodam um seed, ele deverá rodar a autoconfiguração e carregar as migrações no Cloud SQL automaticamente ao iniciar.
3. Faça o login utilizando as variáveis de seed da sua conta administrativa. 

Pronto! A sua aplicação está no ar.
