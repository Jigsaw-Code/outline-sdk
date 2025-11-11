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
    parser = argparse.ArgumentParser(description='Analyze unique domain usage of HTTPS RR parameters.')
    parser.add_argument('input_file', help='Path to the input CSV file.')
    parser.add_argument('output_dir', help='Path to the output directory for plots.')
    args = parser.parse_args()

    # Load the dataset
    try:
        df = pd.read_csv(args.input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {args.input_file}")
        sys.exit(1)

    https_df = df[df['query_type'] == 'HTTPS'].copy()
    https_df.loc[:, 'has_answer'] = https_df['answers'].apply(lambda x: x != '[]')

    # Calculate total unique domains with HTTPS RR
    total_unique_domains_with_https_rr = https_df[https_df['has_answer']]['domain'].nunique()

    # Analysis 2: HTTPS RR Feature Usage (Unique Domains)
    param_domains = {} # Use a dict to store sets for dynamic parameters

    for index, row in https_df[https_df['has_answer']].iterrows():
        try:
            answers = json.loads(row['answers'])
            for answer in answers:
                if 'params' in answer:
                    for param, value in answer['params'].items():
                        # Special handling for 'alpn' to extract individual ALPN values
                        if param == 'alpn' and isinstance(value, list):
                            for alpn_value in value:
                                key = f"alpn:{alpn_value}"
                                if key not in param_domains:
                                    param_domains[key] = set()
                                param_domains[key].add(row['domain'])
                        else: # Handle other parameters
                            key = f"param:{param}" if param not in ['alpn', 'ipv4hint', 'ipv6hint', 'ech'] else param
                            if key not in param_domains:
                                param_domains[key] = set()
                            param_domains[key].add(row['domain'])
        except json.JSONDecodeError:
            continue

    # Convert sets to counts and include dynamically added parameters
    final_param_counts = {}
    for param, domains_set in param_domains.items():
        final_param_counts[param] = len(domains_set)

    # Add total HTTPS RR support to the counts
    final_param_counts['Total HTTPS RR Support'] = total_unique_domains_with_https_rr

    param_df = pd.DataFrame(final_param_counts.items(), columns=['Parameter', 'Unique Domains']).sort_values(by='Unique Domains', ascending=False)

    plt.figure(figsize=(12, 7))
    sns.barplot(x='Unique Domains', y='Parameter', data=param_df)
    plt.title('HTTPS RR Parameter Usage Frequency (Unique Domains)')
    plt.xlabel('Number of Unique Domains')
    plt.ylabel('Parameter')
    plt.tight_layout()
    output_path = os.path.join(args.output_dir, 'param_usage_unique_domains.png')
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
