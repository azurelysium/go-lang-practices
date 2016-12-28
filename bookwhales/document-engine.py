#!/usr/bin/env python2
# -*- coding: utf-8 -*-
#
# Document Engine for Bookwhales
#
import glob
import argparse
import leveldb
from tqdm import tqdm

import MeCab
import unicodedata
from korean import hangul

import numpy
from gensim.models import doc2vec

# Helper functions
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

# Subcommands
def train(args):
    print '>> Train Doc2Vec model for book information'

    db = leveldb.LevelDB(args.database)
    keys = list(db.RangeIter(include_value=False))
    keys = keys[:args.limit]

    print '>> Prepare input data'
    documents = []
    for i in tqdm(xrange(len(keys))):
        # 'content' is str type, not unicode
        isbn, content = keys[i], db.Get(keys[i])
        nouns = extract_noun_only( extract_hangul_only(content.decode('utf8')).encode('utf8') )
        doc = doc2vec.TaggedDocument(nouns, [isbn])
        documents.append(doc)

    print '>> Train, #iter({})'.format(args.iter)
    model = doc2vec.Doc2Vec(alpha=0.025, min_alpha=0.025, workers=args.workers)
    model.build_vocab(documents)
    for epoch in tqdm(xrange(args.iter)):
        model.train(documents)

    print '>> Save model ({})'.format(args.model)
    model.save(args.model)

def __infer_document_vector__(model, content):
    """Infer document vector from plain text
    Args:
        model: d2v model
        content (str, unicode): target text
    """
    nouns = extract_noun_only( extract_hangul_only(content).encode('utf8') )
    return model.infer_vector(nouns)

def __find_similar_by_isbn__(model, isbn):
    tags = model.docvecs.doctags.keys()
    if isbn not in tags:
        return []
    return map(lambda x: x[0], model.docvecs.most_similar(isbn))

def __find_similar_by_content__(model, content):
    vec = __infer_document_vector__(model, content)
    return map(lambda x: x[0], model.docvecs.most_similar([vec]))

def daemon(args):
    print '>> Load model ({})'.format(args.model)
    model = doc2vec.Doc2Vec.load(args.model)
    print __find_similar_by_content__(model, u'여행가서 밥먹는')
    pass

# Main
if __name__ == '__main__':

    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers()

    train_parser = subparsers.add_parser("train")
    train_parser.add_argument('--model', type=str, required=True, help='path for d2v model')
    train_parser.add_argument('--database', type=str, required=True, help='path for LevelDB database')
    train_parser.add_argument('--limit', metavar='N', default=10000, type=int, help='the number of documents to be used for training d2v model')
    train_parser.add_argument('--iter', metavar='N', default=10, type=int, help='the number of iteration for training d2v model')
    train_parser.add_argument('--workers', metavar='N', default=1, type=int, help='the number of cores to use')
    train_parser.set_defaults(func=train)

    daemon_parser = subparsers.add_parser("daemon")
    daemon_parser.set_defaults(func=daemon)
    daemon_parser.add_argument('--model', type=str, required=True, help='path for d2v model')

    args = parser.parse_args()
    args.func(args)
