from jinja2 import Template
from requests import get

def main():
    with open('readme.j2') as fp:
        readme_template = Template(fp.read())

    response = get("https://quotes.rest/qod?category=inspire")
    qotd = 'Unable to retrieve Quote of the Day.'
    if response.status_code == 200:
        qotd = response.json().get('contents').get('quotes')[0].get('quote')
    with open('README.md', 'w') as fp:
        fp.write(readme_template.render(qotd=qotd))

if __name__ == "__main__":
    main()