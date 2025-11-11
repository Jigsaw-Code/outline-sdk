import pandas as pd
import sys
import os

def main():
    if len(sys.argv) < 2:
        print("Usage: python sort_csv_by_rank.py <input_csv_file>")
        sys.exit(1)

    input_file = sys.argv[1]

    try:
        df = pd.read_csv(input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {input_file}")
        sys.exit(1)

    # Sort by 'rank', 'run', and 'query_type'
    df_sorted = df.sort_values(by=['rank', 'run', 'query_type'], kind='mergesort')

    # Create the output filename
    base, ext = os.path.splitext(input_file)
    output_file = f"{base}-sorted{ext}"

    # Write the sorted DataFrame to the new CSV file
    df_sorted.to_csv(output_file, index=False)
    print(f"Successfully sorted {input_file} by rank, run, and query type. Output saved to {output_file}")

if __name__ == '__main__':
    main()