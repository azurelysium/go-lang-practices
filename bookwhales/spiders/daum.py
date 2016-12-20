#
# Spider for crawling book introduction page
# from http://book.daum.net
#
# To run:
# $ scrapy runspider daum.py -a category=KOR05 -o ~/DAUM_KOR05.jsonl -t jsonlines --loglevel=INFO
# $ scrapy runspider daum.py -a category=KOR01 -a leveldb_path=${BOOKWHALES_DATA_ROOT}/DAUM_KOR01 -a stop_duplicate=20 --loglevel=INFO
#
import os
import re
import leveldb
import scrapy
from w3lib.html import remove_tags

class DaumSpider(scrapy.Spider):
    name = 'daum_spider'

    download_delay = 0.5
    autothrottle_enabled = True
    autothrottle_start_delay = 2
    autothrottle_target_concurrency = 1.0

    def __init__(self, category='', leveldb_path=None, stop_duplicate='0', *args, **kwargs):
        super(DaumSpider, self).__init__(*args, **kwargs)

        self.category = category
        self.leveldb = leveldb.LevelDB(leveldb_path)
        self.stop_duplicate = int(stop_duplicate)
        self.n_duplicate = 0

        self.url_template = 'http://book.daum.net/category/book.do?cTab=06&sortType=1&saleStatus=&categoryID={}&pageNo={}&minValue={}&maxValue={}&pageAction={}'
        self.start_urls = [self.url_template.format(self.category, 1, '', '', 0)]

    def __get_or_none_from_leveldb__(self, key):
        try:
            return self.leveldb.Get(key)
        except KeyError:
            return None

    def parse(self, response):
        print 'DaumSpider.parse(): {}'.format(response.url)

        # Extract book item urls
        hrefs = response.css('dt.title a::attr(href)').extract()
        for href in hrefs:
            item_url = 'http://book.daum.net{}'.format(href)
            yield scrapy.Request(response.urljoin(item_url),
                                 callback=self.parse_item)

        # Extract additional information for next page links
        min_value = response.css('#frmSearch input[id="minValue"]::attr(value)').extract_first()
        max_value = response.css('#frmSearch input[id="maxValue"]::attr(value)').extract_first()

        # Generate link for next page
        if len(hrefs) > 0:
            matched = re.search('pageNo=([0-9]+)', response.url)
            if matched:
                page_num = int(matched.group(1)) + 1
                page_action = 1 if page_num % 10 == 1 else 0
                next_page = self.url_template.format(self.category, page_num, min_value, max_value, page_action)
                yield scrapy.Request(response.urljoin(next_page), callback=self.parse)

    def parse_item(self, response):
        print 'DaumSpider.parse_item(): {}'.format(response.url)

        def extract_with_css(query):
            extracted = response.css(query).extract_first()
            return extracted.strip() if extracted else ''

        matched = re.search('ISBN 13-([0-9]+)', response.body)
        if not matched:
            return
        isbn = matched.group(1)

        content = ''
        content += remove_tags(extract_with_css('.introd').encode('utf-8'))
        content += remove_tags(extract_with_css('.authorInfo').encode('utf-8'))
        content += remove_tags(extract_with_css('.book_table').encode('utf-8'))

        if self.leveldb:
            item = self.__get_or_none_from_leveldb__(isbn)
            if item:
                self.n_duplicate += 1
                if (self.stop_duplicate > 0) and (self.n_duplicate >= self.stop_duplicate):
                    raise scrapy.exceptions.CloseSpider('stop_duplicate')
            self.leveldb.Put(isbn, content.encode('utf-8'))
        else:
            yield {'isbn': isbn, 'content': content}
