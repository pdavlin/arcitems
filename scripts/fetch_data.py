#!/usr/bin/env python3
"""
Fetch and aggregate data from RaidTheory/arcraiders-data repository.
Creates bundled JSON files for embedding in the Go binary.
"""

import json
import os
import urllib.request
from datetime import datetime

GITHUB_API = "https://api.github.com/repos/RaidTheory/arcraiders-data/contents"
RAW_URL = "https://raw.githubusercontent.com/RaidTheory/arcraiders-data/main"

def fetch_json(url):
    """Fetch JSON from URL."""
    with urllib.request.urlopen(url) as response:
        return json.loads(response.read().decode())

def fetch_all_files(directory):
    """Fetch all JSON files from a directory in the repo."""
    url = f"{GITHUB_API}/{directory}"
    files = fetch_json(url)

    data = []
    for file_info in files:
        if file_info['type'] == 'file' and file_info['name'].endswith('.json'):
            print(f"Fetching {file_info['path']}...")
            file_url = file_info['download_url']
            file_data = fetch_json(file_url)
            data.append(file_data)

    return data

def main():
    # Create output directory
    os.makedirs('internal/data/bundled', exist_ok=True)

    # Fetch items
    print("\n=== Fetching Items ===")
    items = fetch_all_files('items')
    with open('internal/data/bundled/items.json', 'w') as f:
        json.dump(items, f, indent=2)
    print(f"✓ Saved {len(items)} items")

    # Fetch quests
    print("\n=== Fetching Quests ===")
    quests = fetch_all_files('quests')
    with open('internal/data/bundled/quests.json', 'w') as f:
        json.dump(quests, f, indent=2)
    print(f"✓ Saved {len(quests)} quests")

    # Fetch projects (single file)
    print("\n=== Fetching Projects ===")
    projects_url = f"{RAW_URL}/projects.json"
    projects = fetch_json(projects_url)
    with open('internal/data/bundled/projects.json', 'w') as f:
        json.dump(projects, f, indent=2)
    print(f"✓ Saved projects data")

    # Fetch hideout stations
    print("\n=== Fetching Hideout Stations ===")
    hideouts = fetch_all_files('hideout')
    with open('internal/data/bundled/hideouts.json', 'w') as f:
        json.dump(hideouts, f, indent=2)
    print(f"✓ Saved {len(hideouts)} hideout stations")

    # Create metadata
    metadata = {
        "version": datetime.utcnow().strftime("%Y.%m.%d.%H%M"),
        "syncedAt": datetime.utcnow().isoformat() + "Z",
        "itemCount": len(items),
        "questCount": len(quests),
        "hideoutCount": len(hideouts)
    }
    with open('internal/data/bundled/metadata.json', 'w') as f:
        json.dump(metadata, f, indent=2)
    print(f"✓ Saved metadata (version {metadata['version']})")

    print("\n=== Data fetch complete ===")

if __name__ == '__main__':
    main()
