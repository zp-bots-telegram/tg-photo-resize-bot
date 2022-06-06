FROM python:3-alpine AS build-env
RUN mkdir /install
WORKDIR /bot
RUN apk --no-cache add build-base libffi-dev openssl-dev zlib-dev jpeg-dev
ADD requirements.txt .
RUN pip install --prefix=/install -r requirements.txt

FROM python:3-alpine
RUN apk add libjpeg-turbo
COPY --from=build-env /install /usr/local
WORKDIR /bot
ADD bot.py . 

CMD [ "python", "./bot.py" ]
