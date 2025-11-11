import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import numpy as np
import argparse
import os
import json
from collections import Counter
import sys

def main():
    if len(sys.argv) < 3:
        print("Usage: python unique_domain_analysis.py <input_csv_file> <output_md_file>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]

    try:
        df = pd.read_csv(input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {input_file}")
        sys.exit(1)

    # Filter for successful HTTPS queries
    https_df = df[(df['query_type'] == 'HTTPS') & (df['error'].isna())].copy()

    # Drop duplicates to count each domain only once
    unique_domains_df = https_df.drop_duplicates(subset=['domain'])

    # --- Feature Usage Analysis ---
    alias_mode_count = 0
    alpn_counts = Counter()
    param_counts = Counter()

    for _, row in unique_domains_df.iterrows():
        answers = json.loads(row['answers'])
        for answer in answers:
            if 'HTTPS' in answer:
                https_data = answer['HTTPS']
                
                # AliasMode
                if https_data.get('is_alias', False):
                    alias_mode_count += 1
                
                # ALPN
                if 'alpn' in https_data:
                    for alpn in https_data['alpn']:
                        alpn_counts[alpn] += 1
                
                # Other Parameters
                for param, value in https_data.items():
                    if param not in ['is_alias', 'alpn', 'target_name']:
                        param_counts[param] += 1

    # --- Generate Markdown Table ---
    total_unique_domains = len(unique_domains_df)
    
    markdown_table = "| Feature | Usage Count | Percentage |\n"
    markdown_table += "|:---|:---|:---|"
    
    # AliasMode
    percentage = (alias_mode_count / total_unique_domains) * 100 if total_unique_domains > 0 else 0
    markdown_table += f"| AliasMode | {alias_mode_count} | {percentage:.2f}% |\n"
    
    # ALPN
    for alpn, count in alpn_counts.items():
        percentage = (count / total_unique_domains) * 100 if total_unique_domains > 0 else 0
        markdown_table += f"| alpn={alpn} | {count} | {percentage:.2f}% |\n"
        
    # Other Parameters
    for param, count in param_counts.items():
        percentage = (count / total_unique_domains) * 100 if total_unique_domains > 0 else 0
        markdown_table += f"| {param} | {count} | {percentage:.2f}% |\n"

    with open(output_file, 'w') as f:
        f.write(markdown_table)

if __name__ == '__main__':
    main()
