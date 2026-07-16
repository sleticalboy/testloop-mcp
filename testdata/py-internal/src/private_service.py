class PrivateService:
    def public_value(self, value: str) -> str:
        return self.__normalize(value)

    def __normalize(self, value: str) -> str:
        if value == "":
            return "empty"
        return value.strip().lower()
