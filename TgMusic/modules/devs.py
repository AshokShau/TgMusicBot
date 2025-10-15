#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import ast
import inspect
import io
import os
import platform
import re
import socket
import sys
import traceback
import uuid
from datetime import datetime, timedelta
from html import escape
from sys import version as pyver
from typing import Any, Optional, Tuple, Union

import psutil
from ntgcalls import __version__ as ntgver
from pyrogram import __version__ as pyrover
from pytdbot import Client
from pytdbot import __version__ as py_td_ver
from pytdbot import types
from pytgcalls import __version__ as pytgver

from TgMusic import StartTime
from TgMusic.core import Filter, admins_only, call, chat_cache, config, db
from TgMusic.modules.utils.play_helpers import extract_argument


# We dont modify locals VVVV ; this lets us keep the message available to the user-provided function
async def meval(code, globs, **kwargs):
    # This function is released in the public domain. Feel free to kang it (although I like credit)
    # Note to self: please don't set globals here as they will be lost.
    # Don't clutter locals
    locs = {}
    # Restore globals later
    globs = globs.copy()
    # This code saves __name__ and __package into a kwarg passed to the function.
    # It is set before the users code runs to make sure relative imports work
    global_args = "_globs"
    while global_args in globs.keys():
        # Make sure there's no name collision, just keep prepending _s
        global_args = f"_{global_args}"
    kwargs[global_args] = {}
    for glob in ["__name__", "__package__"]:
        # Copy data to args we are sending
        kwargs[global_args][glob] = globs[glob]

    root = ast.parse(code, "exec")
    code = root.body

    ret_name = "_ret"
    ok = False
    while True:
        if ret_name in globs.keys():
            ret_name = f"_{ret_name}"
            continue
        for node in ast.walk(root):
            if isinstance(node, ast.Name) and node.id == ret_name:
                ret_name = f"_{ret_name}"
                break
            ok = True
        if ok:
            break

    if not code:
        return None

    if not any(isinstance(node, ast.Return) for node in code):
        for i in range(len(code)):
            if isinstance(code[i], ast.Expr) and (
                i == len(code) - 1 or not isinstance(code[i].value, ast.Call)
            ):
                code[i] = ast.copy_location(
                    ast.Expr(
                        ast.Call(
                            func=ast.Attribute(
                                value=ast.Name(id=ret_name, ctx=ast.Load()),
                                attr="append",
                                ctx=ast.Load(),
                            ),
                            args=[code[i].value],
                            keywords=[],
                        )
                    ),
                    code[-1],
                )
    else:
        for node in code:
            if isinstance(node, ast.Return):
                node.value = ast.List(elts=[node.value], ctx=ast.Load())

    code.append(
        ast.copy_location(
            ast.Return(value=ast.Name(id=ret_name, ctx=ast.Load())), code[-1]
        )
    )

    # globals().update(**<global_args>)
    glob_copy = ast.Expr(
        ast.Call(
            func=ast.Attribute(
                value=ast.Call(
                    func=ast.Name(id="globals", ctx=ast.Load()), args=[], keywords=[]
                ),
                attr="update",
                ctx=ast.Load(),
            ),
            args=[],
            keywords=[
                ast.keyword(arg=None, value=ast.Name(id=global_args, ctx=ast.Load()))
            ],
        )
    )
    ast.fix_missing_locations(glob_copy)
    code.insert(0, glob_copy)
    ret_decl = ast.Assign(
        targets=[ast.Name(id=ret_name, ctx=ast.Store())],
        value=ast.List(elts=[], ctx=ast.Load()),
    )
    ast.fix_missing_locations(ret_decl)
    code.insert(1, ret_decl)
    args = []
    for a in list(map(lambda x: ast.arg(x, None), kwargs.keys())):
        ast.fix_missing_locations(a)
        args += [a]
    args = ast.arguments(
        args=[],
        vararg=None,
        kwonlyargs=args,
        kwarg=None,
        defaults=[],
        kw_defaults=[None for _ in range(len(args))],
    )
    args.posonlyargs = []
    fun = ast.AsyncFunctionDef(
        name="tmp", args=args, body=code, decorator_list=[], returns=None
    )
    ast.fix_missing_locations(fun)
    mod = ast.parse("")
    mod.body = [fun]
    comp = compile(mod, "<string>", "exec")

    exec(comp, {}, locs)

    r = await locs["tmp"](**kwargs)
    for i in range(len(r)):
        if hasattr(r[i], "__await__"):
            r[i] = await r[i]  # workaround for 3.5
    i = 0
    while i < len(r) - 1:
        if r[i] is None:
            del r[i]
        else:
            i += 1
    if len(r) == 1:
        [r] = r
    elif not r:
        r = None
    return r


def format_exception(
    exp: BaseException, tb: Optional[list[traceback.FrameSummary]] = None
) -> str:
    """
    Formats an exception traceback as a string, similar to the Python interpreter.
    """

    if tb is None:
        tb = traceback.extract_tb(exp.__traceback__)

    # Replace absolute paths with relative paths
    cwd = os.getcwd()
    for frame in tb:
        if cwd in frame.filename:
            frame.filename = os.path.relpath(frame.filename)

    stack = "".join(traceback.format_list(tb))
    msg = str(exp)
    if msg:
        msg = f": {msg}"

    return f"Traceback (most recent call last):\n{stack}{type(exp).__name__}{msg}"


@Client.on_message(filters=Filter.command("eval"))
@admins_only(only_dev=True)
async def exec_eval(c: Client, m: types.Message) -> None:
    """
    Run python code.
    """
    text = m.text.split(None, 1)
    if len(text) <= 1:
        reply = await m.reply_text("Usage: /eval &lt code &gt")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return None

    code = text[1]
    out_buf = io.StringIO()

    async def _eval() -> Tuple[str, Optional[str]]:
        async def send(
            *args: Any, **kwargs: Any
        ) -> Union["types.Error", "types.Message"]:
            return await m.reply_text(*args, **kwargs)

        def _print(*args: Any, **kwargs: Any) -> None:
            if "file" not in kwargs:
                kwargs["file"] = out_buf
                return print(*args, **kwargs)
            return None

        eval_vars = {
            "loop": c.loop,
            "client": c,
            "stdout": out_buf,
            "c": c,
            "m": m,
            "message": m,
            "msg": m,
            "types": types,
            "send": send,
            "print": _print,
            "inspect": inspect,
            "os": os,
            "re": re,
            "sys": sys,
            "traceback": traceback,
            "uuid": uuid,
            "io": io,
            "db": db,
            "call": call,
        }

        try:
            return "", await meval(code, globals(), **eval_vars)
        except Exception as e:
            first_snip_idx = -1
            tb = traceback.extract_tb(e.__traceback__)
            for i, frame in enumerate(tb):
                if frame.filename == "<string>" or frame.filename.endswith("ast.py"):
                    first_snip_idx = i
                    break

            # Re-raise exception if it wasn't caused by the snippet
            if first_snip_idx == -1:
                raise e

            # Return formatted stripped traceback
            stripped_tb = tb[first_snip_idx:]
            formatted_tb = format_exception(e, tb=stripped_tb)
            return "⚠️ Error:\n\n", formatted_tb

    prefix, result = await _eval()

    if not out_buf.getvalue() or result is not None:
        print(result, file=out_buf)

    out = out_buf.getvalue()
    if out.endswith("\n"):
        out = out[:-1]

    result = f"""{prefix}<b>In:</b>
<pre language="python">{escape(code)}</pre>
<b>ᴏᴜᴛ:</b>
<pre language="python">{escape(out)}</pre>"""

    if len(result) > 2000:
        filename = f"database/{uuid.uuid4().hex}.txt"
        with open(filename, "w", encoding="utf-8") as file:
            file.write(out)

        caption = f"""{prefix}<b>ᴇᴠᴀʟ:</b>
    <pre language="python">{escape(code)}</pre>
    """
        reply = await m.reply_document(
            document=types.InputFileLocal(filename),
            caption=caption,
            disable_notification=True,
            parse_mode="html",
        )
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)

        if os.path.exists(filename):
            os.remove(filename)

        return None

    reply = await m.reply_text(str(result), parse_mode="html")
    if isinstance(reply, types.Error):
        c.logger.warning(reply.message)
    return None


@Client.on_message(filters=Filter.command("stats"))
@admins_only(only_dev=True)
async def sys_stats(client: Client, message: types.Message) -> None:
    """Get comprehensive bot and system statistics including hardware, software, and performance metrics."""
    sys_msg = await message.reply_text(
        f"📊 Gathering <b>{client.me.first_name}</b> system statistics..."
    )
    if isinstance(sys_msg, types.Error):
        client.logger.warning(sys_msg.message)

    # System Information
    hostname = socket.gethostname()
    ip_address = socket.gethostbyname(hostname)
    mac_address = ":".join(re.findall("..", f"{uuid.getnode():012x}"))
    architecture = platform.machine()
    system = platform.system()
    release = platform.release()
    processor = platform.processor() or "Unknown"

    # Hardware Information
    ram = psutil.virtual_memory()
    cores_physical = psutil.cpu_count(logical=False)
    cores_total = psutil.cpu_count(logical=True)

    try:
        cpu_freq = psutil.cpu_freq()
        cpu_freq_str = (
            f"{cpu_freq.current / 1000:.2f} GHz"
            if cpu_freq.current >= 1000
            else f"{cpu_freq.current:.2f} MHz"
        )
        cpu_freq_str += f" (Max: {cpu_freq.max / 1000:.2f} GHz)" if cpu_freq.max else ""
    except Exception as e:
        client.logger.warning("Failed to fetch CPU frequency: %s", e)
        cpu_freq_str = "Unavailable"

    # Disk Information
    disk = psutil.disk_usage("/")
    disk_io = psutil.disk_io_counters()

    # Network Information
    net_io = psutil.net_io_counters()
    net_if = psutil.net_if_addrs()

    # Uptime and Performance
    uptime = timedelta(seconds=int((datetime.now() - StartTime).total_seconds()))
    load_avg = (
        ", ".join([f"{x:.2f}" for x in psutil.getloadavg()])
        if hasattr(psutil, "getloadavg")
        else "N/A"
    )
    cpu_percent = psutil.cpu_percent(interval=1)

    # Database Statistics
    chats = len(await db.get_all_chats())
    users = len(await db.get_all_users())

    def format_bytes(size):
        for unit in ["B", "KiB", "MiB", "GiB", "TiB"]:
            if size < 1024:
                return f"{size:.2f} {unit}"
            size /= 1024
        return f"{size:.2f} PiB"

    response = f"""
<b>⚙️ {client.me.first_name} System Statistics</b>
━━━━━━━━━━━━━━━━━━━━
<b>🕒 Uptime:</b> <code>{uptime}</code>
<b>📈 Load Average:</b> <code>{load_avg}</code>
<b>🧮 CPU Usage:</b> <code>{cpu_percent}%</code>

<b>💬 Database Stats:</b>
  • <b>Chats:</b> <code>{chats:,}</code>
  • <b>Users:</b> <code>{users:,}</code>

<b>📦 Software Versions:</b>
  • <b>Python:</b> <code>{pyver.split()[0]}</code>
  • <b>Pyrogram:</b> <code>{pyrover}</code>
  • <b>Py-TgCalls:</b> <code>{pytgver}</code>
  • <b>NTgCalls:</b> <code>{ntgver}</code>
  • <b>PyTdBot:</b> <code>{py_td_ver}</code>

<b>🖥️ System Information:</b>
  • <b>System:</b> <code>{system} {release}</code>
  • <b>Architecture:</b> <code>{architecture}</code>
  • <b>Processor:</b> <code>{processor}</code>
  • <b>Hostname:</b> <code>{hostname}</code>
  • <b>IP Address:</b> <tg-spoiler>{ip_address}</tg-spoiler>
  • <b>MAC Address:</b> <code>{mac_address}</code>

<b>💾 Memory:</b>
  • <b>RAM:</b> <code>{ram.used / (1024 ** 3):.2f} GiB / {ram.total / (1024 ** 3):.2f} GiB ({ram.percent}%)</code>

<b>🔧 CPU:</b>
  • <b>Cores:</b> <code>{cores_physical} physical, {cores_total} logical</code>
  • <b>Frequency:</b> <code>{cpu_freq_str}</code>

<b>💽 Disk:</b>
  • <b>Total:</b> <code>{disk.total / (1024 ** 3):.2f} GiB</code>
  • <b>Used:</b> <code>{disk.used / (1024 ** 3):.2f} GiB ({disk.percent}%)</code>
  • <b>Free:</b> <code>{disk.free / (1024 ** 3):.2f} GiB</code>
  • <b>IO:</b> <code>Read: {format_bytes(disk_io.read_bytes)}, Write: {format_bytes(disk_io.write_bytes)}</code>

<b>🌐 Network:</b>
  • <b>Sent:</b> <code>{format_bytes(net_io.bytes_sent)}</code>
  • <b>Received:</b> <code>{format_bytes(net_io.bytes_recv)}</code>
  • <b>Interfaces:</b> <code>{len(net_if)} available</code>
"""

    reply = await sys_msg.edit_text(response, disable_web_page_preview=True)
    if isinstance(reply, types.Error):
        client.logger.warning(reply.message)
    return None


@Client.on_message(filters=Filter.command(["activevc", "av"]))
@admins_only(only_dev=True)
async def active_vc(c: Client, message: types.Message) -> None:
    """
    Get active voice chats.
    """
    active_chats = chat_cache.get_active_chats()
    if not active_chats:
        reply = await message.reply_text("No active voice chats.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)

        return None

    text = f"🎵 <b>Active Voice Chats</b> ({len(active_chats)}):\n\n"

    for chat_id in active_chats:
        queue_length = chat_cache.get_queue_length(chat_id)
        if current_song := chat_cache.get_playing_track(chat_id):
            song_info = f"🎶 <b>Now Playing:</b> <a href='{current_song.url}'>{current_song.name}</a> ({current_song.duration}s)"
        else:
            song_info = "🔇 No song playing."

        text += (
            f"➤ <b>Chat ID:</b> <code>{chat_id}</code>\n"
            f"📌 <b>Queue Size:</b> {queue_length}\n"
            f"{song_info}\n\n"
        )

    if len(text) > 4096:
        text = f"🎵 <b>Active Voice Chats</b> ({len(active_chats)})"

    reply = await message.reply_text(text, disable_web_page_preview=True)
    if isinstance(reply, types.Error):
        c.logger.warning(reply.message)
        await message.reply_text(reply.message)
    return None


@Client.on_message(filters=Filter.command("logger"))
@admins_only(only_dev=True)
async def logger(c: Client, message: types.Message) -> None:
    """
    Enable or disable logging.
    """
    if not config.LOGGER_ID or config.LOGGER_ID == 0:
        reply = await message.reply_text("Please set LOGGER_ID in .env first.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return

    args = extract_argument(message.text)
    enabled = await db.get_logger_status(c.me.id)

    if not args:
        status = "enabled ✅" if enabled else "disabled ❌"
        reply = await message.reply_text(
            "Usage: /logger [enable|disable|on|off]\n\nCurrent status: {status}".format(
                status=status
            )
        )
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return

    arg = args.lower()
    if arg in ["on", "enable"]:
        await db.set_logger_status(c.me.id, True)
        reply = await message.reply_text("Logger enabled.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return
    if arg in ["off", "disable"]:
        await db.set_logger_status(c.me.id, False)
        reply = await message.reply_text("Logger disabled.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return

    await message.reply_text(
        "Usage: /logger [enable|disable]\n\nYour argument is {arg}".format(arg=args)
    )


@Client.on_message(filters=Filter.command(["autoend", "auto_end"]))
@admins_only(only_dev=True)
async def auto_end(c: Client, message: types.Message) -> None:
    args = extract_argument(message.text)
    if not args:
        status = await db.get_auto_end(c.me.id)
        status_text = "enabled ✅" if status else "disabled ❌"
        reply = await message.reply_text(
            f"<b>Auto End</b> is currently <b>{status_text}</b>.\n\n"
            "When enabled, the bot will automatically end group voice chats "
            "if no users are listening. Useful for saving resources and keeping chats clean.",
            disable_web_page_preview=True,
        )
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return

    args = args.lower()
    if args in ["on", "enabled"]:
        await db.set_auto_end(c.me.id, True)
        reply = await message.reply_text("✅ <b>Auto End</b> has been <b>enabled</b>.")
    elif args in ["off", "disabled"]:
        await db.set_auto_end(c.me.id, False)
        reply = await message.reply_text("❌ <b>Auto End</b> has been <b>disabled</b>.")
    else:
        reply = await message.reply_text(
            f"⚠️ Unknown argument: <b>{args}</b>\nUse <code>/autoend on</code> or <code>/autoend off</code>.",
            disable_web_page_preview=True,
        )
    if isinstance(reply, types.Error):
        c.logger.warning(reply.message)


@Client.on_message(filters=Filter.command(["clearass", "clearallassistants"]))
@admins_only(only_dev=True)
async def clear_all_assistants(c: Client, message: types.Message) -> None:
    count = await db.clear_all_assistants()
    c.logger.info(
        "Cleared assistants from %s chats by command from %s", count, message.from_id
    )
    reply = await message.reply_text(f"♻️ Cleared assistants from {count} chats")
    if isinstance(reply, types.Error):
        c.logger.warning(reply.message)
    return


@Client.on_message(filters=Filter.command("logs"))
@admins_only(only_dev=True)
async def logs(c: Client, message: types.Message) -> None:
    reply = await message.reply_document(
        document=types.InputFileLocal("bot.log"),
        disable_notification=True,
    )
    if isinstance(reply, types.Error):
        c.logger.warning(reply.message)
