import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import numpy as np
import argparse
import os
import json # Added import

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

    # Chart 1: Quantile Plot of DNS Query Durations by Type (Raw Data)
    plt.figure(figsize=(12, 7))
    quantiles = np.linspace(0, 1, 101)
    for query_type in sorted(df['query_type'].unique()):
        durations = df[df['query_type'] == query_type]['duration_ms']
        if not durations.empty:
            quantile_values = durations.quantile(quantiles)
            sns.lineplot(x=quantiles, y=quantile_values, label=query_type)
    plt.title('Quantile Plot of DNS Query Durations by Type (Raw Data)')
    plt.xlabel('Cumulative Probability')
    plt.ylabel('Duration (ms)')
    plt.ylim(0, 300)
    plt.xticks(np.arange(0, 1.1, 0.1))
    plt.grid(True)
    plt.legend(title='Query Type')
    output_path = os.path.join(args.output_dir, 'duration_by_type_quantile_plot.png') # Renamed output file
    plt.savefig(output_path)
    print(f"Chart 1 saved to {output_path}")
    plt.close()

    # Prepare data for min/median duration plots
    min_durations_per_domain = df.groupby(['domain', 'query_type'])['duration_ms'].min().reset_index()
    median_durations_per_domain = df.groupby(['domain', 'query_type'])['duration_ms'].median().reset_index()

    # Chart 1.1: Quantile Plot of DNS Query Durations by Type (Min per Domain)
    plt.figure(figsize=(12, 7))
    for query_type in sorted(min_durations_per_domain['query_type'].unique()):
        durations = min_durations_per_domain[min_durations_per_domain['query_type'] == query_type]['duration_ms']
        if not durations.empty:
            quantile_values = durations.quantile(quantiles)
            sns.lineplot(x=quantiles, y=quantile_values, label=query_type)
    plt.title('Quantile Plot of DNS Query Durations by Type (Min per Domain)')
    plt.xlabel('Cumulative Probability')
    plt.ylabel('Duration (ms)')
    plt.ylim(0, 300)
    plt.xticks(np.arange(0, 1.1, 0.1))
    plt.grid(True)
    plt.legend(title='Query Type')
    output_path = os.path.join(args.output_dir, 'min_duration_quantile_plot.png')
    plt.savefig(output_path)
    print(f"Chart 1.1 saved to {output_path}")
    plt.close()

    # Chart 1.2: Quantile Plot of DNS Query Durations by Type (Median per Domain)
    plt.figure(figsize=(12, 7))
    for query_type in sorted(median_durations_per_domain['query_type'].unique()):
        durations = median_durations_per_domain[median_durations_per_domain['query_type'] == query_type]['duration_ms']
        if not durations.empty:
            quantile_values = durations.quantile(quantiles)
            sns.lineplot(x=quantiles, y=quantile_values, label=query_type)
    plt.title('Quantile Plot of DNS Query Durations by Type (Median per Domain)')
    plt.xlabel('Cumulative Probability')
    plt.ylabel('Duration (ms)')
    plt.ylim(0, 300)
    plt.xticks(np.arange(0, 1.1, 0.1))
    plt.grid(True)
    plt.legend(title='Query Type')
    output_path = os.path.join(args.output_dir, 'median_duration_quantile_plot.png')
    plt.savefig(output_path)
    print(f"Chart 1.2 saved to {output_path}")
    plt.close()

    # Chart: Parameter Usage (all occurrences)
    https_answers = df[df['query_type'] == 'HTTPS']['answers']
    all_params = []
    for answer_str in https_answers:
        try:
            answers_list = json.loads(answer_str)
            for answer in answers_list:
                if isinstance(answer, dict) and 'params' in answer:
                    for param_key in answer['params'].keys():
                        all_params.append(param_key)
        except json.JSONDecodeError:
            continue # Skip malformed JSON

    if all_params:
        param_counts = pd.Series(all_params).value_counts()
        plt.figure(figsize=(12, 7))
        sns.barplot(x=param_counts.index, y=param_counts.values)
        plt.title('Parameter Usage (All Occurrences)')
        plt.xlabel('SVCB Parameter')
        plt.ylabel('Number of Occurrences')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        output_path = os.path.join(args.output_dir, 'param_usage.png')
        plt.savefig(output_path)
        print(f"Chart: Parameter Usage (All Occurrences) saved to {output_path}")
        plt.close()

if __name__ == '__main__':
    main()
