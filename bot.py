#!/usr/bin/env python3
"""
Bot to reply to uncompressed pictures (below a certain filesize)
with compressed versions and caption of original resolution.
For https://t.me/WallpapersGroup.
"""
import logging
import tempfile
import os
from telegram.ext import Updater, MessageHandler, Filters
from PIL import Image
from pillow_heif import register_heif_opener
from fractions import Fraction
import exifread

# Enable logging
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s -'
                           ' %(message)s from uncompressionbot',
                    level=logging.INFO)

logger = logging.getLogger(__name__)

logger.info("Bot started.")

TYPES = ['image/jpeg', 'image/png', 'image/jpg', 'image/heif']  # Not sure if jpg is us

register_heif_opener()

def document_msg_handler(bot, update):
    msg = update.message
    if msg.document.mime_type not in TYPES:
        return
    if not msg.document.file_size:
        # We don't want to download if we don't know how big it is
        logger.info("Couldn't see filesize of image at {}".format(msg.date))
        return
    if msg.document.file_size > 2.097e+7:  # 20MB
        msg.reply_text('Sorry, image filesize too big, 20MB max')
        return


    file_ext = '.' + msg.document.mime_type.split('/', 1)[1]
    with tempfile.NamedTemporaryFile(suffix=file_ext) as f:
        with tempfile.NamedTemporaryFile(suffix="jpg") as f_send:
            file_t = bot.getFile(msg.document.file_id)
            file_t.download(custom_path=f.name)
            logger.info("Downloaded file from chat {} as {}".format(msg.chat_id, f.name))

            im = Image.open(f.name)
            size_x = str(im.size[0])
            size_y = str(im.size[1])

            logger.info("Got image size = {},{}".format(size_x, size_y))

            if msg.document.file_size > 10194304:  # 10MB
                resize_divider = 2
                while True:
                    new_size_x = int(im.size[0] / resize_divider)
                    new_size_y = int(im.size[1] / resize_divider)
                    logger.info("Converting image to {},{}".format(new_size_x, new_size_y))
                    im_resize = im.resize((new_size_x, new_size_y), Image.ANTIALIAS)
                    im_resize.save(f_send.name, format="jpeg", quality=95, optimise=True)
                    resize_divider *= 2
                    image_size = os.path.getsize(f_send.name)
                    logger.info("Converted image size bytes = {}".format(image_size))
                    if image_size < 10194304 or resize_divider > 8:
                        break
            else:
                im.save(f_send.name, format="jpeg")

            reply = "Original resolution: {}x{}".format(size_x, size_y)

            try:
                reply += get_exif_data(f.name)
            except:
                reply += "; error processing EXIF data :("

            picmsg = msg.reply_photo(f_send, reply)
            logger.debug("Sent reply: {}".format(picmsg))

    logger.info('Replied to chat {} with "{}" and document: {}'.format(msg.chat_id, reply, msg.document))


def get_exif_data(file):
    reply = ""

    with open(file, 'rb') as ff:
        tags = exifread.process_file(ff)
        logger.debug("Got EXIF data = {}".format(tags))
        if tags:
            try:
                reply += "; Model: {} {}".format(tags["Image Make"], tags["Image Model"])
            except KeyError:
                pass

            try:
                reply += "; ISO {}".format(tags["EXIF ISOSpeedRatings"])
            except KeyError:
                pass

            try:
                reply += "; f: {}mm".format(round(float(Fraction(str(tags["EXIF FocalLength"])))))
            except KeyError:
                pass

            try:
                reply += "; f/{}".format(float(Fraction(str(tags["EXIF FNumber"]))))
            except KeyError:
                pass

            try:
                frac = Fraction(str(tags["EXIF ExposureTime"]))
                if frac.denominator > 1000:
                    exposure = str(round(frac, 3))
                else:
                    exposure = str(frac)
                reply += "; {}s".format(exposure)
            except KeyError:
                pass
        else:
            reply += "; no EXIF data :("

    return reply


def error(bot, update, err):
    logger.warning('Update "%s" caused error "%s"' % (update, err))


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
