

import csv
import os
import sys

import matplotlib
matplotlib.use('TkAgg')
import matplotlib.pyplot as plt

def readCsvFile(fileName):
    list_of_dictionaries = []
    print("readCsvFile: fileName = ", fileName)
    with open(fileName, mode ='r') as file:    
        dict_reader = csv.DictReader(file)
        for row in dict_reader:
            list_of_dictionaries.append(row)
        return list_of_dictionaries

def extractField(list_of_dict, field):
    new_list = []
    for line in list_of_dict:
        new_list.append(line[field])
    return new_list

def extract2Fields(list_of_dict, field1, field2):
    list_of_pairs = []
    for line in list_of_dict:
        pair = (line[field1], line[field2])
        list_of_pairs.append(pair)
    return list_of_pairs

def extractFields(list_of_dict, list_of_fields):
    new_list_of_lists = []
    for line in list_of_dict:
        ntuple = []
        for field in list_of_fields:
            ntuple.append(line[field])
        new_list_of_lists.append(ntuple)
    return new_list_of_lists

def getFileNames(dirName):
    fileNames = []
    for entry_name in os.listdir(dirName):
        fileNames.append(entry_name)
    fileNames.sort()
    return fileNames

def graphCPU(dirName):
    fileNames = getFileNames(dirName)
    print("graphCPU: fileNames = ", fileNames)
    for fileName in fileNames:
        fullPath = os.path.join(dirName, fileName)
        csvFileDict = readCsvFile(fullPath)
        time_list = extractField(csvFileDict, 'TimeFromStart')
        time_list_float = [float(p) for p in time_list]
        cpu_perc_list = extractField(csvFileDict, 'CPUPerc')
        cpu_perc_list_float = [float(p.strip('%')) / 100 for p in cpu_perc_list]
        plt.plot(time_list_float, cpu_perc_list_float, label=fileName)
    plt.xlabel("seconds in run")
    plt.ylabel("cpu utilization")
    plt.title("Relationship between time and cpu")
    plt.legend()
    plt.show()

def graphMem(dirName):
    fileNames = getFileNames(dirName)
    print("graphCPU: fileNames = ", fileNames)
    for fileName in fileNames:
        fullPath = os.path.join(dirName, fileName)
        csvFileDict = readCsvFile(fullPath)
        time_list = extractField(csvFileDict, 'TimeFromStart')
        time_list_float = [float(p) for p in time_list]
        memStat = extractField(csvFileDict, 'MemUsage')
        memStatParsed = [float(p.partition('M')[0]) *1024*1024 for p in memStat]
        plt.plot(time_list_float, memStatParsed, label=fileName)
    plt.xlabel("seconds in run")
    plt.ylabel("memory usage")
    plt.title("Relationship between time and memory")
    plt.legend()
    plt.show()

def graphNet(dirName):
    fileNames = getFileNames(dirName)
    print("graphNet: fileNames = ", fileNames)
    for fileName in fileNames:
        fullPath = os.path.join(dirName, fileName)
        csvFileDict = readCsvFile(fullPath)
        time_list = extractField(csvFileDict, 'TimeFromStart')
        time_list_float = [float(p) for p in time_list]
        netStat = extractField(csvFileDict, 'NetIO')
        print("netStat = ", netStat)
        netParsedIn = []
        netParsedOut = []
        for p in netStat:
            inOut = p.split('/')
            inString = inOut[0]
            outString = inOut[1]
            if 'M' in outString:
                netOut = float(outString.partition('M')[0]) *1024*1024
            elif 'G' in outString:
                netOut = float(outString.partition('G')[0]) * 1024 * 1024 * 1024
            elif 'k' in outString:
                netOut = float(outString.partition('k')[0]) * 1024
            else:
                netOut = float(outString)
            if 'M' in inString:
                netIn = float(inString.partition('M')[0]) *1024*1024
            elif 'G' in inString:
                netIn = float(inString.partition('G')[0]) * 1024 * 1024 * 1024
            elif 'k' in inString:
                netIn = float(inString.partition('k')[0]) * 1024
            else:
                netIn = float(inString)
            netParsedOut.append(netOut)
            netParsedIn.append(netIn)

        plt.plot(time_list_float, netParsedIn, label=fileName+' In')
        plt.plot(time_list_float, netParsedOut, label=fileName+' Out')
    plt.xlabel("seconds in run")
    plt.ylabel("network bytes total")
    plt.title("Relationship between time and network")
    plt.legend()
    plt.show()

# main function
if __name__ == "__main__":
    dirName = sys.argv[1]
    graphNet(dirName)

