# Estágio 1: Build
FROM node:18-alpine AS builder

# Define o diretório de trabalho dentro do container
WORKDIR /app

# Copia package.json e package-lock.json primeiro para otimizar o cache do Docker
COPY package*.json ./

# Instala todas as dependências (dev e production)
RUN npm install

# Copia o restante do código da sua aplicação
COPY . .

# Opcional: Se sua aplicação precisa ser compilada (ex: TypeScript), use um comando de build
# RUN npm run build

# Estágio 2: Produção
# Usa uma imagem mais limpa para rodar a aplicação
FROM node:18-alpine

# Define o diretório de trabalho para a aplicação final
WORKDIR /app

# Copia apenas as dependências de produção do estágio anterior
COPY --from=builder /app/node_modules ./node_modules

# Copia o código da aplicação (geralmente os arquivos .js compilados)
COPY --from=builder /app .

# Define o usuário 'node' para segurança
USER node

# Expõe a porta que a aplicação vai rodar
EXPOSE 3000

# Comando para iniciar a aplicação
CMD ["node", "apps/cat/src/app.js"]