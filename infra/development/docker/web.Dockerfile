FROM node:20-alpine

WORKDIR /app

COPY web/package*.json ./

RUN npm install

COPY web ./

RUN npm run build

EXPOSE 5173

CMD ["npm", "run", "dev"]