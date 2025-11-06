import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import numpy as np
import os
import json
from collections import Counter

def main():
    # Define file paths
    report_dir = '/Users/fortuna/firehook/outline-sdk/x/tools/ech-test/report'
    csv_files = [
        os.path.join(report_dir, 'results-top1000-v1.csv'),
        os.path.join(report_dir, 'results-top1000-v2.csv'),
        os.path.join(report_dir, 'results-top1000-v3.csv')
    ]

    # Load and concatenate the datasets
    dfs = []
    for f in csv_files:
        try:
            dfs.append(pd.read_csv(f))
        except FileNotFoundError:
            print(f"Warning: File not found at {f}")
    if not dfs:
        print("Error: No data files found.")
        exit()
    df = pd.concat(dfs, ignore_index=True)

    https_df = df[df['query_type'] == 'HTTPS'].copy()
    https_df.loc[:, 'has_answer'] = https_df['answers'].apply(lambda x: x != '[]')

    # Calculate total unique domains with HTTPS RR
    total_unique_domains_with_https_rr = https_df[https_df['has_answer']]['domain'].nunique()

    # Analysis 2: HTTPS RR Feature Usage (Unique Domains)
    param_domains = {
        'alpn': set(),
        'ipv4hint': set(),
        'ipv6hint': set(),
        'ech': set()
    }

    for index, row in https_df[https_df['has_answer']].iterrows():
        try:
            answers = json.loads(row['answers'])
            for answer in answers:
                if 'params' in answer:
                    for param in answer['params'].keys():
                        if param in param_domains:
                            param_domains[param].add(row['domain'])
        except json.JSONDecodeError:
            continue

    param_counts = {param: len(domains) for param, domains in param_domains.items()}
    
    # Add total HTTPS RR support to the counts
    param_counts['Total HTTPS RR Support'] = total_unique_domains_with_https_rr

    param_df = pd.DataFrame(param_counts.items(), columns=['Parameter', 'Unique Domains']).sort_values(by='Unique Domains', ascending=False)

    plt.figure(figsize=(12, 7))
    sns.barplot(x='Unique Domains', y='Parameter', data=param_df)
    plt.title('HTTPS RR Parameter Usage Frequency (Unique Domains)')
    plt.xlabel('Number of Unique Domains')
    plt.ylabel('Parameter')
    plt.tight_layout()
    output_path = os.path.join(report_dir, 'param_usage_unique_domains.png')
    plt.savefig(output_path)
    print(f"Parameter usage chart (unique domains) saved to {output_path}")
    plt.close()

    # Generate markdown table for param usage
    param_table = "| Parameter | Unique Domains |\n|:---|:---|"
    for index, row in param_df.iterrows():
        param_table += f"| {row['Parameter']} | {row['Unique Domains']} |\n"
    print("\nHTTPS RR Parameter Usage (Unique Domains):\n")
    print(param_table)

if __name__ == '__main__':
    main()
