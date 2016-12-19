#!/usr/bin/env python2
#
# Put an item stored in Json Lines format file to LevelDB
#
import sys
import json
import leveldb
from tqdm import tqdm

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print 'usage: {} <jsonl file> <leveldb path>'.format(sys.argv[0])
        sys.exit()

    db = leveldb.LevelDB(sys.argv[2])
    with open(sys.argv[1]) as f:
        lines = f.readlines()
        for i in tqdm(xrange(len(lines))):
            item = json.loads(lines[i])
            db.Put(item['isbn'], item['content'].encode('utf-8'))
