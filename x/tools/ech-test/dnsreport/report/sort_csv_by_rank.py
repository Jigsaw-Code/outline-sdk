import pandas as pd
import os

def main():
    report_dir = '/Users/fortuna/firehook/outline-sdk/x/tools/ech-test/report'
    csv_file = os.path.join(report_dir, 'results-top1000-n5.csv')

    try:
        df = pd.read_csv(csv_file)
    except FileNotFoundError:
        print(f"Error: File not found at {csv_file}")
        exit()

    # Stable sort by 'rank' column
    df_sorted = df.sort_values(by='rank', kind='mergesort')

    # Write the sorted DataFrame back to the CSV file
    df_sorted.to_csv(csv_file, index=False)
    print(f"Successfully sorted {csv_file} by domain rank.")

if __name__ == '__main__':
    main()