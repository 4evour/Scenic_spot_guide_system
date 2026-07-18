import tempfile
import unittest
from datetime import datetime
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))

from openpyxl import Workbook

from aggregate_consumption import build_output


class ConsumptionAggregationTest(unittest.TestCase):
    def test_build_output_has_all_and_filtered_scopes(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "source.xlsx"
            workbook = Workbook()
            sheet = workbook.active
            sheet.append(
                [
                    "tourist_id",
                    "user_nickname",
                    "age",
                    "gender",
                    "attraction_name",
                    "attraction_content",
                    "attraction_type",
                    "visit_date",
                    "stay_duration",
                    "ticket_cost",
                    "food_cost",
                    "shopping_cost",
                    "transport_cost",
                    "entertainment_cost",
                    "total_cost",
                    "group_size",
                    "satisfaction",
                ]
            )
            sheet.append(
                [
                    "U1",
                    "a",
                    30,
                    "女",
                    "灵山大佛",
                    "灵山官方资料",
                    "文化",
                    datetime(2025, 1, 1),
                    3,
                    100,
                    20,
                    10,
                    5,
                    15,
                    150,
                    2,
                    5,
                ]
            )
            sheet.append(
                [
                    "U2",
                    "b",
                    40,
                    "男",
                    "其他景点",
                    "其他资料",
                    "自然",
                    datetime(2025, 2, 1),
                    2,
                    80,
                    10,
                    0,
                    5,
                    5,
                    100,
                    1,
                    4,
                ]
            )
            workbook.save(source)

            result = build_output(source, "灵山")

        self.assertEqual(result["all"]["summary"]["sample_count"], 2)
        self.assertEqual(result["lingshan"]["summary"]["sample_count"], 1)
        self.assertEqual(result["lingshan"]["summary"]["total_cost"], 150.0)
        self.assertEqual(result["all"]["source_metadata"]["source_row_count"], 2)


if __name__ == "__main__":
    unittest.main()
