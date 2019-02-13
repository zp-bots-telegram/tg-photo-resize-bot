#!/usr/bin/env python3
"""
Bot to reply to uncompressed pictures (below a certain filesize)
with compressed versions and caption of original resolution.
For https://t.me/WallpapersGroup.
"""
import logging
import tempfile
import os
from telegram.ext import Updater, CommandHandler, MessageHandler, Filters
from telegram import ParseMode
from PIL import Image


# Enable logging
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s -'
                    ' %(message)s from uncompressionbot',
                    level=logging.INFO)

logger = logging.getLogger(__name__)

TYPES = ['image/jpeg','image/png','image/jpg'] # Not sure if jpg is used
TG_PHOTO_GROUP = os.environ.get('TG_PHOTO_GROUP')
TG_PHOTO_CHANNEL = os.environ.get('TG_PHOTO_CHANNEL')


def document_msg_handler(bot, update):
    msg = update.message
    if msg.document.mime_type not in TYPES:
        return
    if not msg.document.file_size:
        # We don't want to download if we don't know how big it is
        logger.info("Couldn't see filesize of image at {}".format(msg.date))
        return
    if msg.document.file_size > 10194304: # 4MB
        msg.reply_text('Sorry, image filesize too big')
        return
    file_ext = '.' + msg.document.mime_type.split('/',1)[1]
    with tempfile.NamedTemporaryFile(suffix=file_ext) as f:
        file_t = bot.getFile(msg.document.file_id)
        file_t.download(custom_path=f.name)
        im = Image.open(f.name)
        logger.debug("Downloaded file")
        size_x = str(im.size[0])
        size_y = str(im.size[1])
        logger.debug("Got image size = {},{}".format(size_x,size_y))
        print(f.name)
        picmsg = msg.reply_photo(f,
                                 "Original resolution: {}x{}".format(size_x,size_y))
    if TG_PHOTO_CHANNEL is not None and update.effective_chat.id == TG_PHOTO_GROUP:
        # Forward the uncompressed and compressed wallpapers to a channel
        bot.forwardMessage(TG_PHOTO_CHANNEL,
                           message_id=msg.message_id,
                           from_chat_id=TG_PHOTO_GROUP)
        bot.forwardMessage(TG_PHOTO_CHANNEL,
                           message_id=picmsg.message_id,
                           from_chat_id=TG_PHOTO_GROUP)
    print(msg.document)


def error(bot, update, error):
    logger.warn('Update "%s" caused error "%s"' % (update, error))


def main():
    # Create the EventHandler and pass it the bot token
    updater = Updater(os.environ.get('TG_BOT_KEY'))

    dp = updater.dispatcher

    dp.add_handler(MessageHandler(Filters.document, document_msg_handler))

    # Log all errors
    dp.add_error_handler(error)

    # Start the bot
    updater.start_polling()

    # Idle this thread
    updater.idle()


if __name__ == "__main__":
    main()
