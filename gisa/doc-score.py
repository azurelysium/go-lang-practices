#!/usr/bin/env python2
#
# Document scoring script for news articles
#
import argparse
import sqlite3
from tqdm import tqdm

import MeCab
import unicodedata
from korean import hangul

import numpy
from gensim.models import doc2vec

DATABASE = '/home/azurelysium/Documents/articles.db'
D2V_MODEL = '/home/azurelysium/Documents/articles.d2v'

def extract_hangul_only(unistr):
    unistr = unistr.replace('\n', ' ')
    hangul_str = []
    for i in xrange(len(unistr)):
        try:
            uniname = unicodedata.name(unistr[i])
            if ('HANGUL' in uniname) or ('SPACE' in uniname) or ('FULL STOP' in uniname):
                hangul_str.append(unistr[i])
        except ValueError:
            pass
    return ''.join(hangul_str)

def apply_mecab(txt):
    m = MeCab.Tagger('-d /usr/local/lib/mecab/dic/mecab-ko-dic')
    output = m.parse(txt)
    output = output.split('\n')
    output = [item.split(',') for item in output]
    tagged = []
    for line in output:
        if line[0]=='' or line[0]=='EOS':
            continue
        tagged.append(line[0])
    return tagged

def extract_noun_only(txt):
    tagged = apply_mecab(txt)
    tagged = map(lambda x: x.split('\t'), tagged)
    tagged = filter(lambda x: len(x)>1 and x[1][:2] == 'NN', tagged)
    tagged = map(lambda x: x[0], tagged)
    return tagged

def get_documents(extra_clause):
    documents = []
    with sqlite3.connect(DATABASE) as conn:
        c = conn.cursor()
        for row in c.execute('SELECT id, title, content FROM articles ' + extra_clause):
            document = (row[0], ' '.join(row[1:]))
            documents.append(document)
    return documents

def get_all_documents(limit):
    return get_documents('WHERE ignored = 0 LIMIT {}'.format(limit))
def get_read_documents(read=True):
    return get_documents('WHERE ignored = 0 AND read = {}'.format(1 if read else 0))

def train(args):
    print '>> Train Doc2Vec model for news articles'
    docs = get_all_documents(args.limit)
    print '#docs({})'.format(len(docs))

    print '>> Prepare input data'
    documents = []
    for i in tqdm(range(len(docs))):
        article_id, text = docs[i]
        nouns = extract_noun_only( extract_hangul_only(text).encode('utf8') )
        doc = doc2vec.TaggedDocument(nouns, [article_id])
        documents.append(doc)

    print '>> Train, #iter({})'.format(args.iter)
    model = doc2vec.Doc2Vec(alpha=0.025, min_alpha=0.025, workers=1)
    model.build_vocab(documents)
    for epoch in tqdm(range(args.iter)):
        model.train(documents)

    print '>> Save model ({})'.format(D2V_MODEL)
    model.save(D2V_MODEL)

def score(args):
    print '>> Score news articles using Doc2Vec model'
    docs_read = get_read_documents()
    print '#docs_read({})'.format(len(docs_read))

    print '>> Load model ({})'.format(D2V_MODEL)
    model = doc2vec.Doc2Vec.load(D2V_MODEL)
    tags = model.docvecs.doctags.keys()

    print '>> Infer vectors'
    vectors = []
    for i in tqdm(range(len(docs_read))):
        article_id, text = docs_read[i]
        nouns = extract_noun_only( extract_hangul_only(text).encode('utf8') )
        vectors.append(model.infer_vector(nouns))

    print '>> Compute an average vector'
    mean_vector = numpy.mean(vectors, axis=0)

    print '>> Score articles'
    docs_unread = get_read_documents(False)
    print '#docs_unread({})'.format(len(docs_unread))

    with sqlite3.connect(DATABASE) as conn:
        c = conn.cursor()

        updates = []
        for i in tqdm(range(len(docs_unread))):
            article_id, text = docs_unread[i]
            nouns = extract_noun_only( extract_hangul_only(text).encode('utf8') )
            vector = model.infer_vector(nouns)

            distance = 1e-10 + numpy.linalg.norm(vector - mean_vector)
            score = 1. / distance
            updates.append((score, article_id))

        c.executemany('UPDATE articles SET score = ? WHERE id = ?', updates)
        conn.commit()


if __name__ == '__main__':

    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers()

    train_parser = subparsers.add_parser("train")
    train_parser.add_argument('--limit', metavar='N', default=10000, type=int, help='the number of documents to be used for training d2v model')
    train_parser.add_argument('--iter', metavar='N', default=10, type=int, help='the number of iteration for training d2v model')
    train_parser.set_defaults(func=train)

    score_parser = subparsers.add_parser("score")
    score_parser.set_defaults(func=score)

    args = parser.parse_args()
    args.func(args)
