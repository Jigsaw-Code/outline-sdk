import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import numpy as np
import argparse
import os

def get_tld_category(domain):
    parts = domain.split('.')
    if len(parts) > 1:
        tld = parts[-1]
        if len(tld) == 2:
            return tld
    return 'other'

def main():
    parser = argparse.ArgumentParser(description='Generate performance report charts.')
    parser.add_argument('input_file', help='Path to the input CSV file.')
    parser.add_argument('output_dir', help='Path to the output directory.')
    args = parser.parse_args()

    # Load the dataset
    try:
        df = pd.read_csv(args.input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {args.input_file}")
        exit()

    # Chart 1: Quantile Plot of DNS Query Durations by Type
    plt.figure(figsize=(12, 7))
    quantiles = np.linspace(0, 1, 101)
    for query_type in sorted(df['query_type'].unique()):
        durations = df[df['query_type'] == query_type]['duration_ms']
        if not durations.empty:
            quantile_values = durations.quantile(quantiles)
            sns.lineplot(x=quantiles, y=quantile_values, label=query_type)
    plt.title('Quantile Plot of DNS Query Durations by Type')
    plt.xlabel('Cumulative Probability')
    plt.ylabel('Duration (ms)')
    plt.ylim(0, 300)
    plt.xticks(np.arange(0, 1.1, 0.1))
    plt.grid(True)
    plt.legend(title='Query Type')
    output_path = os.path.join(args.output_dir, 'quantile_plot.png')
    plt.savefig(output_path)
    print(f"Chart 1 saved to {output_path}")
    plt.close()

    # Prepare data for the next charts
    a_queries = df[df['query_type'] == 'A'][['domain', 'duration_ms']]
    https_queries = df[df['query_type'] == 'HTTPS'][['domain', 'duration_ms']]
    merged_df = pd.merge(a_queries, https_queries, on='domain', suffixes=('_A', '_HTTPS'))
    merged_df['duration_diff'] = merged_df['duration_ms_HTTPS'] - merged_df['duration_ms_A']
    merged_df['tld_category'] = merged_df['domain'].apply(get_tld_category)

    # Chart 2: Distribution of Duration Difference (HTTPS - A) by TLD Category
    plt.figure(figsize=(15, 8))
    sns.boxplot(x='tld_category', y='duration_diff', data=merged_df, showfliers=False)
    plt.xticks(rotation=90)
    plt.title('Distribution of Duration Difference (HTTPS - A) by TLD Category')
    plt.xlabel('TLD Category')
    plt.ylabel('Duration Difference (ms)')
    plt.grid(True)
    plt.tight_layout()
    output_path = os.path.join(args.output_dir, 'duration_diff_by_tld.png')
    plt.savefig(output_path)
    print(f"Chart 2 saved to {output_path}")
    plt.close()

    # Chart 3: Histogram of HTTPS Query Durations
    plt.figure(figsize=(12, 7))
    sns.histplot(https_queries['duration_ms'], bins=50, kde=True)
    plt.title('Distribution of HTTPS Query Durations')
    plt.xlabel('Duration (ms)')
    plt.ylabel('Frequency')
    plt.xlim(0, 1000)
    plt.grid(True)
    output_path = os.path.join(args.output_dir, 'https_duration_histogram.png')
    plt.savefig(output_path)
    print(f"Chart 3 saved to {output_path}")
    plt.close()

if __name__ == '__main__':
    main()
