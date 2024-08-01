from matplotlib import pyplot as plt
import matplotlib
import csv
import subprocess
import os
import itertools
from matplotlib import cycler
import json
import numpy as np

matplotlib.rcParams["figure.figsize"] = [4, 3]

DEFAULT_COLORS = [
    "#1f77b4",
    "#ff7f0e",
    "#2ca02c",
    "#d62728",
    "#9467bd",
    "#8c564b",
    "#e377c2",
    "#7f7f7f",
    "#bcbd22",
    "#17becf",
]

TRACES = ["syn", "borg"]
# TRACES = ["syn"]

INFINITE_TENANTS = 10000
STANDARD_RAs = {"borg": 95, "syn": 90}
XLIMs = {"borg": (10, 30), "syn": (180, 210)}


def main():
    # subprocess.check_call("go test -v ./eval/...", shell=True)
    os.makedirs("figs", exist_ok=True)
    funcs = [
        segmented_multiplier,
        mtadv_LC_vs_RA,
        mtadv_LC_vs_TC,
        stadv_lc_vs_time,
        benign_free_duration_cdf_all,
        benign_lc_vs_time,
        benign_LC_vs_RA,
        benign_allocation_duration_cdf,
        fourier,
        latex_stats,
    ]

    for func in funcs:
        print(func.__name__)
        try:
            func()
        except FileNotFoundError as e:
            print(f"    Skipping due to missing data: {e.filename}")


def read_jsonl(name):
    l = []
    with open(name) as f:
        for line in f:
            l.append(json.loads(line))
    return l


def fourier():
    fig, ax = plt.subplots()
    with open("figs/fourier.csv") as f:
        r = csv.reader(f)
        points = [(i, float(x), float(y)) for i, x, y in r]

    for id in sorted({i for i, x, y in points}):
        xs = [x * 24 for i, x, y in points if i == id]
        ys = [y for i, x, y in points if i == id]

        ax.plot(xs, ys, color=("b" if id == "9" else "0.80"))

    ax.set_xlim(0, 24)
    ax.set_ylim(50, 100)
    ax.set_xticks([0, 6, 12, 18, 24])
    ax.set_xlabel("Time (hour of day)")
    ax.set_ylabel("Target instance count")
    fig.tight_layout()

    fig.savefig("figs/fourier.pdf")


LABEL_SORT_KEYS = {"random": 0, "fifo": 1, "tagged": 2, "segmented": 3, "segmented-neg": 4}
LABELS = {
    "random": "Rᴀɴᴅᴏᴍ",
    "fifo": "LRU",
    "tagged": "Tᴀɢɢᴇᴅ",
    "segmented": "Sᴇɢᴍᴇɴᴛᴇᴅ",
    "segmented-neg": "Sᴇɢᴍᴇɴᴛᴇᴅ-NEG",
}


def plot_over_time(dataset, metric):
    fig, (ax1, ax2) = plt.subplots(
        2, 1, gridspec_kw={"height_ratios": [5, 1]}, figsize=(4, 3)
    )
    ax1.set_prop_cycle(
        cycler(linestyle=["-", "--", ":", "-."], color=DEFAULT_COLORS[:4])
    )
    data = dataset
    for series in sorted(data, key=lambda x: LABEL_SORT_KEYS[x["Policy"]["Type"]]):
        label = series["Policy"]["Type"]
        time_series = sorted(
            (float(t), stat) for t, stat in series["TimeSeriesStats"].items()
        )
        day_time_series = [t / 3600 / 24 for t, _ in time_series]
        ax1.plot(
            day_time_series,
            [metric(stat) for _, stat in time_series],
            label=LABELS[label],
        )

    ax1.set_xlim(0, 10)

    ax1.set_xticks([])
    ax1.set_xticks(
        range(0, 100, 2),
        labels=None,
        minor=True,
    )

    Ra_series = data[0]
    label = Ra_series["Policy"]["Type"]
    time_series = sorted(
        (float(t), stat) for t, stat in Ra_series["TimeSeriesStats"].items()
    )
    ax2.plot(
        [t / 3600 / 24 for t, _ in time_series],
        [1 - stat["availableIPs"] / Ra_series["TotalIPs"] for _, stat in time_series],
        label=LABELS[label],
        color="k",
    )

    ax2.set_xlim(0, 10)
    ax2.set_yticks([])
    ax2.set_xlabel("Time (days)")
    ax2.set_ylabel("$AR_t$")
    ax1.grid(
        which="minor",
        axis="x",
        color="0.9",
    )
    ax2.grid(
        which="minor",
        axis="x",
        color="0.9",
    )

    # ax1.legend(loc="upper center", ncol=2)

    return fig, (ax1, ax2)


def plot_parameter_space(dataset, metric, parameter):
    fig, ax = plt.subplots()
    ax.set_prop_cycle(
        cycler(linestyle=["-", "--", ":", "-."], color=DEFAULT_COLORS[:4])
    )
    data = dataset
    for label in LABEL_SORT_KEYS.keys():
        series = list(
            sorted(
                [d for d in data if d["Policy"]["Type"] == label],
                key=lambda d: parameter(d),
            )
        )
        xs = [parameter(d) for d in series]
        ys = [metric(d) for d in series]

        ax.plot(xs, ys, label=LABELS[label])
    return fig, ax


def benign_lc_vs_time():
    for name in TRACES:
        data = read_jsonl(f"figs/{name}-benign.jsonl")
        data = [
            d
            for d in data
            if d["OverallStats"]["targetAllocRatio"] == STANDARD_RAs[name]
        ]
        fig, (ax1, ax2) = plot_over_time(
            data, lambda stat: stat["latentConf"] / stat["allocated"]
        )
        ax1.set_ylabel("Latent config probability")
        ax1.set_xlim(0, 10)
        ax2.set_xlim(0, 10)
        fig.tight_layout()
        fig.subplots_adjust(hspace=0, wspace=0)
        fig.savefig(f"figs/{name}-benign-lc_vs_time.pdf")
        plt.close(fig)


def benign_LC_vs_RA():
    for name in TRACES:
        fig, ax = plot_parameter_space(
            read_jsonl(f"figs/{name}-benign.jsonl"),
            metric=lambda d: d["OverallStats"]["latentConf"]
            / d["OverallStats"]["allocated"],
            parameter=lambda d: d["OverallStats"]["targetAllocRatio"] / 100,
        )
        ax.set_xlabel("Max pool utilization ($AR_{max}$)")
        ax.set_ylabel("Latent conf probability")
        ax.set_xlim(0.5, 1.0)
        ax.grid(axis="x", color="0.9")
        fig.tight_layout()

        fig.savefig(f"figs/{name}-benign-LC_vs_RA.pdf")
        plt.close(fig)


def benign_allocation_duration_cdf():
    for name in TRACES:
        fig, ax = plt.subplots(figsize=(4, 2))

        xs = read_jsonl(f"figs/{name}-benign.jsonl")[0]["OverallStats"][
            "allocationDurationCDF"
        ]
        ys = np.linspace(0, 1, len(xs))
        ax.plot(xs, ys)

        ax.set_xlabel("Allocation duration (seconds)")
        ax.set_ylabel("CDF")
        ax.set_xscale("log")
        fig.tight_layout()

        fig.savefig(f"figs/{name}-allocation_duration_cdf.pdf")
        plt.close(fig)


def benign_free_duration_cdf_all():
    for name in TRACES:
        fig, axs = plt.subplots(2, 1)
        for i, allocRatio in enumerate([80, STANDARD_RAs[name]]):
            axs[i].set_prop_cycle(
                cycler(linestyle=["-", "--", ":", "-."], color=DEFAULT_COLORS[:4])
            )
            data = read_jsonl(f"figs/{name}-benign.jsonl")
            data = [
                d for d in data if d["OverallStats"]["targetAllocRatio"] == allocRatio
            ]
            for series in sorted(
                data, key=lambda x: LABEL_SORT_KEYS[x["Policy"]["Type"]]
            ):
                label = series["Policy"]["Type"]
                xs = series["OverallStats"]["freeDurationCDF"]
                ys = np.linspace(0, 1, len(xs))
                axs[i].plot(xs, ys, label=LABELS[label])

                axs[i].set_ylabel(f"CDF\n($AR_{'{max}'}={allocRatio/100}$)")
                axs[i].set_xscale("log")
                axs[i].set_xlim(1800, 100000)
        axs[1].set_xlabel("Free duration (seconds)")

        # ax.legend()

        fig.tight_layout()
        legend = axs[0].legend(loc="upper center", ncol=4, bbox_to_anchor=(0.5, 2))

        fig.savefig(f"figs/{name}-free_duration_cdf_all.pdf")
        export_legend(legend, f"figs/legend.pdf")
        plt.close(fig)


def stadv_lc_vs_time():
    for name in TRACES:
        for nt, nt_label in [(1, "stadv"), (INFINITE_TENANTS, "mtadv")]:
            for objective, ob_label in [
                ("newLatentConfs", "Latent config yield"),
                ("newUniques", "Unique IP Yield"),
            ]:
                data = read_jsonl(f"figs/{name}-adv.jsonl")
                data = [
                    d
                    for d in data
                    if d["OverallStats"]["targetAllocRatio"] == STANDARD_RAs[name]
                    and d["OverallStats"]["numAdversaryTenants"] == nt
                ]
                fig, (ax1, ax2) = plot_over_time(
                    data,
                    lambda stat: stat["adversary"]
                    and stat["adversary"][objective] / stat["adversary"]["created"],
                )
                ax1.set_ylabel(ob_label)
                ax1.set_xlim(*XLIMs[name])
                ax2.set_xlim(*XLIMs[name])
                fig.tight_layout()
                fig.subplots_adjust(hspace=0, wspace=0)
                fig.savefig(f"figs/{name}-{nt_label}-{objective}_vs_time.pdf")
                plt.close(fig)


def mtadv_LC_vs_TC():
    for name in TRACES:
        for objective, ob_label, file_name in [
                (lambda d: d["OverallStats"]["adversary"]["newLatentConfs"]
                    / d["OverallStats"]["adversary"]["created"], "Latent config yield", "newLatentConfs"),
                (lambda d: d["OverallStats"]["adversary"]["totalUniques"]
                    / d["OverallStats"]["adversary"]["created"], "Unique IP Yield", "totalUniques"),
                    (lambda d: d["OverallStats"]["adversary"]["adversaryBenignExploitedAllocs"]
                    / d["OverallStats"]["adversary"]["adversaryBenignAllocs"], "Poison Rate", "poisonRate")
            ]:
            data = [
                d
                for d in read_jsonl(f"figs/{name}-adv.jsonl")
                if d["OverallStats"]["targetAllocRatio"] == STANDARD_RAs[name]
            ]
            fig, ax = plot_parameter_space(
                data,
                metric=objective,
                parameter=lambda d: d["OverallStats"]["numAdversaryTenants"],
            )
            ax.set_xlabel("Num Adv Tenants")
            ax.set_ylabel(ob_label)
            ax.set_xscale("log")
            ax.grid(axis="x", color="0.9")
            ax.set_xlim(1, 10000)
            fig.tight_layout()

            fig.savefig(f"figs/{name}-{file_name}_vs_TC.pdf")
            plt.close(fig)


def mtadv_LC_vs_RA():
    for name in TRACES:
        for nt, nt_label in [(1, "stadv"), (INFINITE_TENANTS, "mtadv")]:
            for objective, ob_label, file_name in [
                (lambda d: d["OverallStats"]["adversary"]["newLatentConfs"]
                    / d["OverallStats"]["adversary"]["created"], "Latent config yield", "newLatentConfs"),
                (lambda d: d["OverallStats"]["adversary"]["totalUniques"]
                    / d["OverallStats"]["adversary"]["created"], "Unique IP Yield", "totalUniques"),
                    # (lambda d: d["OverallStats"]["adversary"]["adversaryBenignExploitedAllocs"]
                    # / d["OverallStats"]["adversary"]["adversaryBenignAllocs"], "Poison Rate", "poisonRate")
            ]:
                data = [
                    d
                    for d in read_jsonl(f"figs/{name}-adv.jsonl")
                    if d["OverallStats"]["numAdversaryTenants"] == nt
                ]
                fig, ax = plot_parameter_space(
                    data,
                    metric=objective,
                    parameter=lambda d: d["OverallStats"]["targetAllocRatio"] / 100,
                )
                plt.plot([0],[0], alpha=0)
                ax.set_xlabel("Max pool utilization ($AR_{max}$)")
                ax.set_ylabel(ob_label)
                ax.set_xlim(0.5, 1)
                ax.grid(axis="x", color="0.9")
                fig.tight_layout()

                fig.savefig(f"figs/{name}-{nt_label}-{file_name}_vs_RA.pdf")
                plt.close(fig)


def latex_stats():
    
    for name in TRACES:
        data = [
                d
                for d in read_jsonl(f"figs/{name}-adv.jsonl")
            ]
        for objective, ob_label in [
            ("newLatentConfs", "Latent config yield"),
            ("totalUniques", "Unique IP Yield"),
        ]:
            metric=lambda d: d["OverallStats"]["adversary"][objective]/ d["OverallStats"]["adversary"]["created"]
            for new, old in [("segmented","tagged"), ("segmented", "random"), ("tagged", "random")]:
                newVal = metric([d for d in data if d["OverallStats"]["numAdversaryTenants"] == INFINITE_TENANTS and
                            d["OverallStats"]["targetAllocRatio"] == STANDARD_RAs[name] and
                            d["Policy"]["Type"] == new
                ][0])
                oldVal = metric([d for d in data if d["OverallStats"]["numAdversaryTenants"] == INFINITE_TENANTS and
                            d["OverallStats"]["targetAllocRatio"] == STANDARD_RAs[name] and
                            d["Policy"]["Type"] == old
                ][0])
                print(rf"\newcommand{{\best{name}mtadv{new}{objective}ImprovementOver{old}}}{{\SI{{{100*(1-newVal/oldVal):.1f}}}{{\%}}}}")
                # print(f"best{name}mtadv{new}{objective}ImprovementOver{old} = {100*(1-newVal/oldVal)}")

def segmented_multiplier():
    fig, ax = plt.subplots()
    ax.set_prop_cycle(
        cycler(linestyle=["-", "--", ":", "-."], color=DEFAULT_COLORS[:4])
    )
    data = read_jsonl("figs/segmented_multipliers.jsonl")
    series = list(
        sorted(
            [d for d in data if d["OverallStats"]["multiplier"] > 0],
            key=lambda d: d["OverallStats"]["multiplier"],
        )
    )
    xs = [d["OverallStats"]["multiplier"] for d in series]
    ys = [
        d["OverallStats"]["adversary"]["newLatentConfs"]
        / d["OverallStats"]["adversary"]["created"]
        for d in series
    ]
    print("segmentedLCYieldVariation", 100*(1-min(ys)/max(ys)))

    ax.plot(xs, [y/max(ys) for y in ys], label="Latent Config Yield")

    xs = [d["OverallStats"]["multiplier"] for d in series]
    ys = [
        d["OverallStats"]["adversary"]["totalUniques"]
        / d["OverallStats"]["adversary"]["created"]
        for d in series
    ]
    print("segmentedIPYieldVariation", 100*(1-min(ys)/max(ys)))

    ax.plot(xs, [y/max(ys) for y in ys], label="Unique IP Yield")
    ax.legend()
    # plt.yscale("log")
    ax.grid(axis="x", color="0.9")
    ax.set_xlim(0, 5)
    ax.set_xticks(range(0, 5))
    ax.set_xlabel("Segmentation Cooldown Multiplier $\\alpha$")
    ax.set_ylabel("Metric (Normalized)")
    fig.tight_layout()

    fig.savefig("figs/segmented_multipliers.pdf")
    plt.close(fig)


def export_legend(legend, filename="legend.png"):
    fig = legend.figure
    fig.canvas.draw()
    bbox = legend.get_window_extent().transformed(fig.dpi_scale_trans.inverted())
    fig.savefig(filename, dpi="figure", bbox_inches=bbox)


if __name__ == "__main__":
    main()
