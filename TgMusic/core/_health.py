from aiohttp import web
import asyncio

from typing import TYPE_CHECKING
if TYPE_CHECKING:
    from TgMusic import Bot

class HealthCheck:
    def __init__(self, client: 'Bot', port, host='0.0.0.0'):
        self.client = client
        self.port = port
        self.host = host
        self.app = web.Application()
        self.runner = None
        self.site = None

    async def home(self, _: web.Request):
        bot_name = self.client.me.first_name
        return web.json_response({
            'Bot': bot_name,
            'version': getattr(self.client, '_version', 'unknown'),
            'uptime': getattr(self.client, '_get_uptime', lambda: 0)()
        })

    async def health_check(self, _: web.Request):
        import logging

        if not self.client:
            logging.error("HealthCheck failed: Client not initialized")
            raise web.HTTPServiceUnavailable(text="Service temporarily unavailable")

        if not getattr(self.client, 'is_running', False):
            logging.error("HealthCheck failed: Client not running")
            raise web.HTTPServiceUnavailable(text="Service temporarily unavailable")

        try:
            await self.client.call.health_check()
        except RuntimeError as e:
            self.client.logger.error(f"Health check failed: {e}")
            raise web.HTTPServiceUnavailable(text="Pyrogram client not running")

        return web.json_response({
            'status': 'healthy',
            'version': getattr(self.client, '_version', 'unknown'),
            'uptime': getattr(self.client, '_get_uptime', lambda: 0)(),
            'timestamp': asyncio.get_event_loop().time(),
        })

    async def start(self):
        self.app.router.add_get('/', self.home)
        self.app.router.add_get('/health', self.health_check)
        self.runner = web.AppRunner(self.app)
        await self.runner.setup()
        self.site = web.TCPSite(self.runner, self.host, self.port)
        await self.site.start()
        self.client.logger.info(f"Health check server started on http://{self.host}:{self.port}")

    async def stop(self):
        if self.site:
            await self.site.stop()
        if self.runner:
            await self.runner.cleanup()
