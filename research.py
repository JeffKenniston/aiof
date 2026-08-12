import urllib.request
import json
import re

url = "https://en.wikipedia.org/w/api.php?action=query&prop=extracts&exintro&titles=Cannabis&format=json"
req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
with urllib.request.urlopen(req) as response:
    data = json.loads(response.read().decode())

pages = data['query']['pages']
for page_id in pages:
    extract = pages[page_id]['extract']
    # clean html tags
    clean_text = re.sub('<[^<]+>', '', extract)
    with open('marijuana_research.txt', 'w') as f:
        f.write(clean_text)
    print("Research saved to marijuana_research.txt")
