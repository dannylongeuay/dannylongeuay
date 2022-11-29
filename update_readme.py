import logging
from datetime import datetime, timezone
from jinja2 import Template
from requests import get


def main():
    with open("readme.j2") as fp:
        readme_template = Template(fp.read())

    response = get("https://zenquotes.io/api/today")
    response.raise_for_status()

    quotes = response.json()
    quote = quotes.pop()
    qotd = quote.get("h", "Unable to retrieve Quote of the Day.")
    timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S %Z")

    with open("README.md", "w") as fp:
        fp.write(readme_template.render(qotd=qotd, timestamp=timestamp))

    logging.info("updated README.md")


if __name__ == "__main__":
    main()
