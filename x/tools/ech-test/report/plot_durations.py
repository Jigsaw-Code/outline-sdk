import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import numpy as np
import argparse

def main():
    parser = argparse.ArgumentParser(description='Generate a quantile plot of DNS query durations.')
    parser.add_argument('input_file', help='Path to the input CSV file.')
    parser.add_argument('output_file', help='Path to the output PNG file.')
    args = parser.parse_args()

    # Load the dataset
    try:
        df = pd.read_csv(args.input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {args.input_file}")
        exit()

    # Create a figure and axes for the plot
    plt.figure(figsize=(12, 7))

    # Define the quantiles to compute
    quantiles = np.linspace(0, 1, 101)

    # Calculate and plot for each query type
    for query_type in sorted(df['query_type'].unique()):
        durations = df[df['query_type'] == query_type]['duration_ms']
        if not durations.empty:
            quantile_values = durations.quantile(quantiles)
            sns.lineplot(x=quantiles, y=quantile_values, label=query_type)


    # Set plot title and labels
    plt.title('Quantile Plot of DNS Query Durations by Type')
    plt.xlabel('Cumulative Probability')
    plt.ylabel('Duration (ms)')
    plt.ylim(0, 300)  # Set y-axis limit
    plt.xticks(np.arange(0, 1.1, 0.1))
    plt.grid(True)
    plt.legend(title='Query Type')


    # Save the plot to a file
    plt.savefig(args.output_file)

    print(f"Plot saved to {args.output_file}")

if __name__ == '__main__':
    main()
