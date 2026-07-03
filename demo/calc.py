"""demo Python file for testing the pytest test generator"""


def add(a, b):
    return a + b


def divide(a, b):
    if b == 0:
        raise ValueError("division by zero")
    return a / b


def format_text(text: str, prefix: str = "", *args, **kwargs) -> str:
    return prefix + text + "".join(args)


async def fetch_data(url: str) -> dict:
    import aiohttp

    async with aiohttp.ClientSession() as session:
        async with session.get(url) as response:
            return await response.json()


class Calculator:
    def __init__(self):
        self.history = []

    def add(self, a, b):
        result = a + b
        self.history.append(result)
        return result

    def divide(self, a, b):
        if b == 0:
            raise ValueError("division by zero")
        return a / b

    def clear(self):
        self.history = []

    @staticmethod
    def version():
        return "1.0.0"
