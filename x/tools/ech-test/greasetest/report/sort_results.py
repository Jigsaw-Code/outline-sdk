import pandas as pd

file_path = '/Users/fortuna/firehook/outline-sdk/x/tools/ech-test/greasetest/report/grease-results-top1000.csv'
df = pd.read_csv(file_path)

df_sorted = df.sort_values(by=['rank', 'ech_grease'], ascending=[True, True])

df_sorted.to_csv(file_path, index=False)

print(f"Sorted the file: {file_path}")