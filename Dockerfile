FROM python:3
WORKDIR /bot
ADD requirements.txt .
ADD bot.py . 
RUN pip install -r requirements.txt
CMD [ "python", "./bot.py" ]
